// Command seed-price-history gives every demo product a month of price changes.
//
// A product page shows its change history and a 30-day price chart, both built
// from the audit log. Seeding through the API works, but every change is then
// stamped with the moment the script ran: the chart becomes a flat line with
// every point piled up at the right-hand edge, which demonstrates nothing.
//
// This writes the same history spread across the past month. The events are
// produced by the real audit service and signed with the real keys, so the
// chain verifies exactly as any other change does -- only the moment each one
// claims to have happened is chosen. That is what demo data is: the shops and
// the products are invented too.
//
// It rewrites the product history it finds, so point it at a demo database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	mongorepo "vngrocery/internal/repository/mongo"
	auditservice "vngrocery/internal/service/audit"
	"vngrocery/pkg/config"
	mongopkg "vngrocery/pkg/mongodb"
	vaultpkg "vngrocery/pkg/vault"
)

// priceMove is a reason a seller changes a price, and the direction it moves.
// A history of bare numbers says nothing; the note is what makes it legible.
type priceMove struct {
	ratio float64
	note  string
}

var priceMoves = []priceMove{
	{0.06, "Nguồn cung giảm do mưa kéo dài."},
	{-0.05, "Vào vụ, hàng về nhiều nên hạ giá."},
	{0.04, "Chi phí vận chuyển tăng."},
	{-0.03, "Điều chỉnh theo giá chợ đầu mối."},
	{0.05, "Hàng tuyển loại 1, giá nhập cao hơn."},
	{-0.04, "Khuyến mãi cuối tuần."},
	{0.03, "Giá đầu vào nhích lên."},
	{-0.02, "Cân đối lại theo nhu cầu."},
}

func main() {
	days := flag.Int("days", 30, "how far back the history should reach")
	changes := flag.Int("changes", 5, "price changes per product")
	seed := flag.Int64("seed", 20260821, "makes the generated history repeatable")
	flag.Parse()

	if *days < 2 || *changes < 1 || *changes > len(priceMoves) {
		log.Fatalf("days must be at least 2 and changes between 1 and %d", len(priceMoves))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	if !cfg.UseMongo() {
		log.Fatalf("MONGODB_ENABLED must be true: MongoDB is the only supported storage backend")
	}
	if !cfg.VaultEnabled {
		log.Fatalf("VAULT_ENABLED must be true: the events have to be signed for real")
	}

	ctx := context.Background()
	app, err := mongopkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize MongoDB: %v", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			log.Printf("failed to close MongoDB resources: %v", closeErr)
		}
	}()

	vaultClient := vaultpkg.NewClient(vaultpkg.Config{
		Address:        cfg.VaultAddr,
		Token:          cfg.VaultToken,
		KVMount:        cfg.VaultKVMount,
		KeysPathPrefix: cfg.VaultKeysPathPrefix,
	})

	products := mongorepo.NewProductRepository(app.Database)
	events := mongorepo.NewEventLogRepository(app.Database)
	authUsers := mongorepo.NewAuthUserRepository(app.Database)

	// The clock is swapped per event so each one is stamped with the moment it
	// claims to have happened.
	var clock time.Time
	audit := auditservice.
		NewService(events, authUsers, vaultClient).
		WithClock(func() time.Time { return clock })

	all, err := products.List(ctx, repository.ProductListFilter{})
	if err != nil {
		log.Fatalf("failed to list products: %v", err)
	}

	random := rand.New(rand.NewSource(*seed))
	written := 0
	for _, product := range all {
		schedule := timestamps(*changes, *days, random)
		if err := rewrite(ctx, events, audit, products, &clock, product, schedule); err != nil {
			log.Printf("  %s: %v", product.Name, err)
			continue
		}
		written++
		fmt.Printf("  %-28s %d thay đổi trong %d ngày\n", trim(product.Name, 28), *changes, *days)
	}

	fmt.Printf("\n%d/%d sản phẩm đã có lịch sử giá\n", written, len(all))
}

// timestamps spreads the changes across the window, oldest first, with the
// creation at the start. Real spacing is uneven, so this is too.
func timestamps(changes, days int, random *rand.Rand) []time.Time {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -days)
	span := now.Sub(start)

	points := make([]time.Time, 0, changes+1)
	points = append(points, start)
	for i := 1; i <= changes; i++ {
		// Evenly spaced, then nudged by up to a third of a slot so the line
		// does not look mechanically regular.
		slot := span / time.Duration(changes+1)
		jitter := time.Duration(random.Int63n(int64(slot / 3)))
		points = append(points, start.Add(slot*time.Duration(i)+jitter))
	}
	return points
}

// rewrite replaces one product's recorded history with a backdated one.
func rewrite(
	ctx context.Context,
	events *mongorepo.EventLogRepository,
	audit *auditservice.Service,
	products repository.ProductRepository,
	clock *time.Time,
	product domain.Product,
	schedule []time.Time,
) error {
	// The old chain has to go: an event's hash covers the one before it, so a
	// backdated entry cannot simply be appended to entries stamped later.
	if _, err := events.DeleteByResource(ctx, "product", product.ProductID); err != nil {
		return fmt.Errorf("failed to clear the old history: %w", err)
	}

	// Walk backwards from today's price so the product ends where it started,
	// rather than drifting to whatever the generated moves happen to produce.
	prices := backwardsFrom(product.Price, len(schedule)-1)

	current := product
	current.Price = prices[0]
	current.CreatedAt = schedule[0]
	current.UpdatedAt = schedule[0]
	current.Version = 1

	*clock = schedule[0]
	if err := audit.Log(ctx, auditservice.Input{
		ActorUserID:     product.OwnerUserID,
		ResourceType:    "product",
		ResourceID:      product.ProductID,
		ResourceVersion: current.Version,
		Action:          "product.created",
		Status:          "created",
		Payload:         auditservice.MutationPayload{After: current},
	}); err != nil {
		return fmt.Errorf("failed to record the creation: %w", err)
	}

	for i := 1; i < len(schedule); i++ {
		before := current
		current.Price = prices[i]
		current.FreshnessNote = priceMoves[(i-1)%len(priceMoves)].note
		current.UpdatedAt = schedule[i]
		current.Version = i + 1

		*clock = schedule[i]
		if err := audit.Log(ctx, auditservice.Input{
			ActorUserID:     product.OwnerUserID,
			ResourceType:    "product",
			ResourceID:      product.ProductID,
			ResourceVersion: current.Version,
			Action:          "product.updated",
			Status:          "updated",
			Payload:         auditservice.MutationPayload{Before: before, After: current},
		}); err != nil {
			return fmt.Errorf("failed to record change %d: %w", i, err)
		}
	}

	// The product itself has to agree with the last thing its history says.
	return products.Save(ctx, current)
}

// backwardsFrom builds a price series ending at final, so the listing keeps the
// price it already had and the history explains how it got there.
func backwardsFrom(final float64, changes int) []float64 {
	prices := make([]float64, changes+1)
	prices[changes] = final

	for i := changes - 1; i >= 0; i-- {
		move := priceMoves[i%len(priceMoves)]
		// Undo the move that led to the next price.
		previous := prices[i+1] / (1 + move.ratio)
		prices[i] = round(previous)
	}
	return prices
}

// round snaps to the nearest 1.000d: a price tag is not 68.317d.
func round(value float64) float64 {
	snapped := math.Round(value/1000) * 1000
	if snapped < 1000 {
		return 1000
	}
	return snapped
}

func trim(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width-1]) + "…"
}
