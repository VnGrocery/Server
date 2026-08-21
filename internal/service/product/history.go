package product

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

// ShortSHALength is how much of a hash is shown in a list, the way a commit is
// quoted by its first few characters rather than all sixty-four.
const ShortSHALength = 6

// DefaultPriceWindowDays is how far back the price chart looks.
const DefaultPriceWindowDays = 30

// HistoryEntry is one recorded change to a product.
//
// The audit log already chains every mutation: each entry carries the SHA-256
// of its own content and the id of the one before it, signed with the actor's
// key. That is the same shape as a commit history, and this exposes it as one.
type HistoryEntry struct {
	// SHA of the entry's content, in hex. Stored base64; hex is what people
	// expect to see and to paste into a search box.
	SHA         string
	ShortSHA    string
	PreviousSHA string

	Sequence    int
	Action      string
	Status      string
	ActorUserID string
	ActorName   string
	OccurredAt  time.Time

	// Verified is true when the content hash, the signature and the link to the
	// previous entry all check out.
	Verified         bool
	ContentHashValid bool
	SignatureValid   bool
	ChainLinkValid   bool

	// PriceAfter is the price this change left in place. Nil when the entry did
	// not carry a product body (a deletion, say).
	PriceAfter *float64

	// Changes lists the fields this entry altered, empty for the first entry.
	Changes []FieldChange
}

func (e *HistoryEntry) applyVerdict(verdict audit.EventVerificationResult) {
	e.ContentHashValid = verdict.ContentHashValid
	e.SignatureValid = verdict.SignatureValid
	e.ChainLinkValid = verdict.ChainLinkValid
	e.Verified = verdict.Verified
}

// FieldChange is one field's before and after, rendered as text so the client
// does not have to know the shape of every field the product has.
type FieldChange struct {
	Field  string
	Before string
	After  string
}

// PricePoint is the price in effect at a moment.
type PricePoint struct {
	At    time.Time
	Price float64
}

// ProductHistory is the change history and the price series derived from it.
type ProductHistory struct {
	ProductID string

	// Entries newest first, the way a commit list reads.
	Entries []HistoryEntry

	// ChainVerified is true only when every entry verified. A single failure
	// means the record has been altered since it was written.
	ChainVerified bool

	// PriceHistory covers the requested window. The first point carries the
	// price already in effect when the window opened, so a chart does not start
	// from nothing when the last change was months ago.
	PriceHistory []PricePoint

	WindowDays int

	// Market is what every other shop charges for the same product. Zero-valued
	// when nothing else sells it, which is most products.
	Market MarketPrice
}

// productSnapshot is the product body as the audit log marshals it: Go field
// names, because it stores the domain struct directly. Decoding it here keeps
// that detail out of the API.
type productSnapshot struct {
	Name           string   `json:"Name"`
	Description    string   `json:"Description"`
	Category       string   `json:"Category"`
	Price          float64  `json:"Price"`
	Currency       string   `json:"Currency"`
	FreshnessScore float64  `json:"FreshnessScore"`
	FreshnessNote  string   `json:"FreshnessNote"`
	Status         string   `json:"Status"`
	Tags           []string `json:"Tags"`
	ImageURLs      []string `json:"ImageURLs"`
}

type mutationSnapshot struct {
	Before *productSnapshot `json:"before"`
	After  *productSnapshot `json:"after"`
}

// HistoryInput asks for one product's history.
type HistoryInput struct {
	ShopID     string
	ProductID  string
	WindowDays int
}

// History reconstructs a product's change history from the audit log.
func (s *Service) History(ctx context.Context, input HistoryInput) (ProductHistory, error) {
	productID := strings.TrimSpace(input.ProductID)
	if productID == "" {
		return ProductHistory{}, fmt.Errorf("%w: productId is required", ErrInvalidProduct)
	}
	if s.events == nil {
		return ProductHistory{}, fmt.Errorf("event log repository is not configured")
	}

	// Confirms the product exists and belongs to the shop in the path, so a
	// guessed id cannot be used to read another shop's history.
	product, err := s.GetByID(ctx, strings.TrimSpace(input.ShopID), productID)
	if err != nil {
		return ProductHistory{}, err
	}

	events, err := s.events.List(ctx, repository.EventLogListFilter{
		ResourceType: "product",
		ResourceID:   productID,
	})
	if err != nil {
		return ProductHistory{}, err
	}

	// Re-checks every hash, signature and chain link. Without this the flags on
	// each entry would be decoration: the point of the list is that it can be
	// shown to have not been altered.
	verdicts := s.verifyChain(ctx, productID)

	// Oldest first while building, so each entry can be compared with the one
	// before it; reversed at the end for display.
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })

	windowDays := input.WindowDays
	if windowDays <= 0 {
		windowDays = DefaultPriceWindowDays
	}

	entries := make([]HistoryEntry, 0, len(events))
	points := make([]PricePoint, 0, len(events))
	chainVerified := true

	var previous *productSnapshot
	for _, event := range events {
		snapshot := decodeMutation(event.PayloadJSON)
		entry := s.historyEntry(ctx, event, previous, snapshot)
		entry.applyVerdict(verdicts[event.EventID])
		if !entry.Verified {
			chainVerified = false
		}
		entries = append(entries, entry)

		if snapshot.After != nil {
			points = append(points, PricePoint{
				At:    entry.OccurredAt,
				Price: snapshot.After.Price,
			})
			previous = snapshot.After
		}
	}

	// Newest first for display.
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}

	// A product with no recorded changes still has a price; without this the
	// chart would be empty for every product nobody has edited yet.
	if len(points) == 0 {
		points = append(points, PricePoint{At: product.CreatedAt, Price: product.Price})
	}

	// A shop's own price means little on its own; what makes it checkable is
	// what everyone else charges for the same thing.
	market, err := s.Market(ctx, product, windowDays)
	if err != nil {
		return ProductHistory{}, err
	}

	return ProductHistory{
		ProductID:     productID,
		Entries:       entries,
		ChainVerified: len(events) > 0 && chainVerified,
		PriceHistory:  priceWindow(points, s.now(), windowDays),
		WindowDays:    windowDays,
		Market:        market,
	}, nil
}

// verifyChain returns each event's verdict by id. An empty map when there is no
// verifier configured, which leaves every entry unverified rather than claiming
// a verification that did not happen.
func (s *Service) verifyChain(ctx context.Context, productID string) map[string]audit.EventVerificationResult {
	if s.verifier == nil {
		return nil
	}
	result, err := s.verifier.VerifyResource(ctx, audit.VerifyResourceInput{
		ResourceType: "product",
		ResourceID:   productID,
	})
	if err != nil {
		return nil
	}

	verdicts := make(map[string]audit.EventVerificationResult, len(result.Events))
	for _, event := range result.Events {
		verdicts[event.EventID] = event
	}
	return verdicts
}

func (s *Service) historyEntry(
	ctx context.Context,
	event domain.EventLog,
	previous *productSnapshot,
	snapshot mutationSnapshot,
) HistoryEntry {
	sha := hexSHA(event.ContentSHA256)

	entry := HistoryEntry{
		SHA:         sha,
		ShortSHA:    shorten(sha),
		Sequence:    event.Sequence,
		Action:      event.Action,
		Status:      event.Status,
		ActorUserID: event.ActorUserID,
		ActorName:   s.actorName(ctx, event.ActorUserID),
		OccurredAt:  parseOccurredAt(event),
	}

	if snapshot.After != nil {
		price := snapshot.After.Price
		entry.PriceAfter = &price
	}

	// The payload's own "before" is authoritative when present; otherwise the
	// previous entry's "after" is what this change replaced.
	was := snapshot.Before
	if was == nil {
		was = previous
	}
	entry.Changes = diffSnapshots(was, snapshot.After)

	return entry
}

// actorName resolves who made the change. Empty when it cannot be looked up --
// the entry is still worth showing without a name.
func (s *Service) actorName(ctx context.Context, userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" || s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(user.DisplayName)
}

func decodeMutation(payloadJSON string) mutationSnapshot {
	var snapshot mutationSnapshot
	if strings.TrimSpace(payloadJSON) == "" {
		return snapshot
	}
	// A payload this code cannot read is not a reason to hide the entry: the
	// hash, the signature and the timestamp are all still meaningful.
	_ = json.Unmarshal([]byte(payloadJSON), &snapshot)
	return snapshot
}

// hexSHA converts the stored base64 digest to hex. Falls back to the original
// text when it is not base64, so an unexpected format is shown rather than
// swallowed.
func hexSHA(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return stored
	}
	return hex.EncodeToString(raw)
}

func shorten(sha string) string {
	if len(sha) <= ShortSHALength {
		return sha
	}
	return sha[:ShortSHALength]
}

func parseOccurredAt(event domain.EventLog) time.Time {
	if at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(event.OccurredAt)); err == nil {
		return at
	}
	return event.CreatedAt
}

// priceWindow trims the series to the last windowDays, keeping one point at the
// window's start carrying whatever price was already in effect.
func priceWindow(points []PricePoint, now time.Time, windowDays int) []PricePoint {
	if len(points) == 0 {
		return nil
	}
	sort.Slice(points, func(i, j int) bool { return points[i].At.Before(points[j].At) })

	start := now.AddDate(0, 0, -windowDays)
	inside := make([]PricePoint, 0, len(points))
	carried := PricePoint{}
	hasCarried := false

	for _, point := range points {
		if point.At.Before(start) {
			carried = PricePoint{At: start, Price: point.Price}
			hasCarried = true
			continue
		}
		inside = append(inside, point)
	}

	if hasCarried {
		inside = append([]PricePoint{carried}, inside...)
	}
	return inside
}

// diffSnapshots lists what changed between two versions of a product.
//
// Rendered as text because the client only needs to show it: a chart reads the
// price separately, and everything else is a label in a list.
func diffSnapshots(before, after *productSnapshot) []FieldChange {
	if after == nil || before == nil {
		return nil
	}

	changes := make([]FieldChange, 0, 4)
	add := func(field, was, now string) {
		if was != now {
			changes = append(changes, FieldChange{Field: field, Before: was, After: now})
		}
	}

	add("name", before.Name, after.Name)
	add("description", before.Description, after.Description)
	add("category", before.Category, after.Category)
	add("price", formatAmount(before.Price), formatAmount(after.Price))
	add("currency", before.Currency, after.Currency)
	add("freshnessScore", formatScore(before.FreshnessScore), formatScore(after.FreshnessScore))
	add("freshnessNote", before.FreshnessNote, after.FreshnessNote)
	add("status", before.Status, after.Status)
	add("tags", strings.Join(before.Tags, ", "), strings.Join(after.Tags, ", "))
	add("images", formatCount(len(before.ImageURLs)), formatCount(len(after.ImageURLs)))

	return changes
}

// formatAmount prints a price without a trailing ".00", so a change reads as
// "68000 -> 72000" rather than "68000.000000 -> 72000.000000".
func formatAmount(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatScore(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatCount(count int) string { return strconv.Itoa(count) }
