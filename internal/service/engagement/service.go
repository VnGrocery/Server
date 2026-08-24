package engagement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

var (
	ErrInvalidTarget = errors.New("target must be a shop or a product")
	ErrInvalidKind   = errors.New("kind must be follow, like or love")
	ErrNotFound      = errors.New("target not found")
)

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

// Anchorer puts the new total in the queue for the chain. It is optional: with
// no chain configured the marks still count, they just carry no proof.
type Anchorer interface {
	PrepareEngagementCount(count domain.EngagementCount) (domain.EngagementCount, error)
}

type Service struct {
	engagements repository.EngagementRepository
	shops       repository.ShopRepository
	products    repository.ProductRepository
	audit       AuditLogger
	anchorer    Anchorer
	now         func() time.Time
}

func NewService(
	engagements repository.EngagementRepository,
	shops repository.ShopRepository,
	products repository.ProductRepository,
	auditLogger AuditLogger,
	anchorer Anchorer,
) *Service {
	return &Service{
		engagements: engagements,
		shops:       shops,
		products:    products,
		audit:       auditLogger,
		anchorer:    anchorer,
		now:         time.Now,
	}
}

// State is what one reader sees: the totals, which of them they put there, and
// where the totals were last written down on chain.
type State struct {
	TargetType string
	TargetID   string

	Follows int
	Likes   int
	Loves   int

	Mine []string

	ChainTxHash      string
	ChainBlockNumber int64
	ChainAnchorTime  *time.Time
	AnchorStatus     string
	DataHash         string
}

// kindsFor is which marks a target can carry. A shop cannot be loved and a
// product cannot be followed, so the other counts are never queried and never
// printed as a zero that looks like nobody cared.
func kindsFor(targetType string) []string {
	switch targetType {
	case domain.EngagementTargetShop:
		return []string{domain.EngagementFollow}
	case domain.EngagementTargetProduct:
		return []string{domain.EngagementLike, domain.EngagementLove}
	default:
		return nil
	}
}

func validKind(targetType, kind string) bool {
	for _, allowed := range kindsFor(targetType) {
		if allowed == kind {
			return true
		}
	}
	return false
}

func countID(targetType, targetID string) string {
	return targetType + ":" + targetID
}

func markID(userID, targetType, targetID, kind string) string {
	return strings.Join([]string{userID, targetType, targetID, kind}, ":")
}

// Toggle adds the mark or takes it back, and returns what the target looks like
// afterwards. The second tap is a removal rather than an error: the button the
// reader pressed is the same button either way.
func (s *Service) Toggle(ctx context.Context, userID, targetType, targetID, kind string) (State, error) {
	userID = strings.TrimSpace(userID)
	targetType = strings.TrimSpace(targetType)
	targetID = strings.TrimSpace(targetID)
	kind = strings.TrimSpace(kind)

	if kindsFor(targetType) == nil {
		return State{}, ErrInvalidTarget
	}
	if !validKind(targetType, kind) {
		return State{}, ErrInvalidKind
	}
	if userID == "" || targetID == "" {
		return State{}, ErrInvalidTarget
	}
	if err := s.requireTarget(ctx, targetType, targetID); err != nil {
		return State{}, err
	}

	id := markID(userID, targetType, targetID, kind)
	had, err := s.engagements.Has(ctx, id)
	if err != nil {
		return State{}, err
	}

	action := "engagement.added"
	if had {
		action = "engagement.removed"
		if err := s.engagements.Delete(ctx, id); err != nil {
			return State{}, err
		}
	} else {
		if err := s.engagements.Save(ctx, domain.Engagement{
			EngagementID: id,
			UserID:       userID,
			TargetType:   targetType,
			TargetID:     targetID,
			Kind:         kind,
			CreatedAt:    s.now().UTC(),
		}); err != nil {
			return State{}, err
		}
	}

	count, err := s.recount(ctx, targetType, targetID)
	if err != nil {
		return State{}, err
	}
	if err := s.log(ctx, userID, count, kind, action); err != nil {
		return State{}, err
	}

	return s.state(ctx, userID, targetType, targetID, count)
}

// Get reads the totals without changing them. userID may be empty, for a reader
// who is not signed in and can only look.
func (s *Service) Get(ctx context.Context, userID, targetType, targetID string) (State, error) {
	targetType = strings.TrimSpace(targetType)
	targetID = strings.TrimSpace(targetID)
	if kindsFor(targetType) == nil {
		return State{}, ErrInvalidTarget
	}
	if targetID == "" {
		return State{}, ErrInvalidTarget
	}

	count, err := s.engagements.GetCount(ctx, countID(targetType, targetID))
	if err != nil {
		return State{}, err
	}
	// A target nobody has marked has no stored record. Its totals are zero,
	// which is a fact worth serving rather than an error.
	if strings.TrimSpace(count.CountID) == "" {
		count.TargetType = targetType
		count.TargetID = targetID
	}
	return s.state(ctx, strings.TrimSpace(userID), targetType, targetID, count)
}

func (s *Service) requireTarget(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case domain.EngagementTargetShop:
		if s.shops == nil {
			return nil
		}
		if _, err := s.shops.GetByID(ctx, targetID); err != nil {
			return fmt.Errorf("%w: %s", ErrNotFound, targetID)
		}
	case domain.EngagementTargetProduct:
		if s.products == nil {
			return nil
		}
		if _, err := s.products.GetByID(ctx, targetID); err != nil {
			return fmt.Errorf("%w: %s", ErrNotFound, targetID)
		}
	}
	return nil
}

// recount counts the marks rather than adding one to a stored figure, so a
// total can never drift away from the rows it is supposed to be counting.
func (s *Service) recount(ctx context.Context, targetType, targetID string) (domain.EngagementCount, error) {
	count, err := s.engagements.GetCount(ctx, countID(targetType, targetID))
	if err != nil {
		return domain.EngagementCount{}, err
	}

	now := s.now().UTC()
	if strings.TrimSpace(count.CountID) == "" {
		count = domain.EngagementCount{
			CountID:    countID(targetType, targetID),
			TargetType: targetType,
			TargetID:   targetID,
			CreatedAt:  now,
		}
	}

	for _, kind := range kindsFor(targetType) {
		total, err := s.engagements.CountKind(ctx, targetType, targetID, kind)
		if err != nil {
			return domain.EngagementCount{}, err
		}
		switch kind {
		case domain.EngagementFollow:
			count.Follows = total
		case domain.EngagementLike:
			count.Likes = total
		case domain.EngagementLove:
			count.Loves = total
		}
	}

	count.Version++
	count.UpdatedAt = now
	if s.anchorer != nil {
		count, err = s.anchorer.PrepareEngagementCount(count)
		if err != nil {
			return domain.EngagementCount{}, err
		}
	}
	if err := s.engagements.SaveCount(ctx, count); err != nil {
		return domain.EngagementCount{}, err
	}
	return count, nil
}

func (s *Service) state(ctx context.Context, userID, targetType, targetID string, count domain.EngagementCount) (State, error) {
	state := State{
		TargetType:       targetType,
		TargetID:         targetID,
		Follows:          count.Follows,
		Likes:            count.Likes,
		Loves:            count.Loves,
		Mine:             []string{},
		ChainTxHash:      count.ChainTxHash,
		ChainBlockNumber: count.ChainBlockNumber,
		ChainAnchorTime:  count.ChainAnchorTime,
		AnchorStatus:     count.ChainAnchorStatus,
		DataHash:         count.DataHash,
	}
	if userID == "" {
		return state, nil
	}
	mine, err := s.engagements.ListKindsByUser(ctx, userID, targetType, targetID)
	if err != nil {
		return State{}, err
	}
	state.Mine = mine
	return state, nil
}

// log signs the tap itself. The chain carries the total; this carries who moved
// it and when, so a figure that jumps can be read back to the taps behind it.
func (s *Service) log(ctx context.Context, userID string, count domain.EngagementCount, kind, action string) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     userID,
		ResourceType:    "engagement",
		ResourceID:      count.CountID,
		ResourceVersion: count.Version,
		Action:          action,
		Status:          kind,
		Payload: map[string]any{
			"targetType": count.TargetType,
			"targetId":   count.TargetID,
			"kind":       kind,
			"follows":    count.Follows,
			"likes":      count.Likes,
			"loves":      count.Loves,
		},
	})
}
