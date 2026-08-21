package product

import (
	"context"
	"sort"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

// MarketPrice is what every shop selling the same product charges for it.
//
// Built to answer one question: is this shop's price in line with everyone
// else's? So it carries the average alongside the spread, not just a single
// number that could hide a shop charging double.
type MarketPrice struct {
	// CatalogKey the comparison was made on, so a client can show what was
	// treated as "the same product".
	CatalogKey string

	// ShopCount includes this shop. One means nobody else sells it and there is
	// nothing to compare against.
	ShopCount int

	CurrentAverage float64
	CurrentLowest  float64
	CurrentHighest float64

	// History is the average price in effect over the window, one point per
	// moment any of the shops changed a price.
	History []PricePoint
}

// Comparable reports whether there is anything to compare against. One shop
// selling something is not a market, and drawing an "average" line identical to
// the shop's own price would say nothing while looking like it said something.
func (m MarketPrice) Comparable() bool { return m.ShopCount > 1 }

// marketSeries is one shop's price timeline for the shared product.
type marketSeries struct {
	shopID string
	points []PricePoint
}

// Market builds the cross-shop price comparison for one product.
func (s *Service) Market(ctx context.Context, product domain.Product, windowDays int) (MarketPrice, error) {
	key := CatalogKey(product.Name, product.Category)
	if key == "" || s.products == nil {
		return MarketPrice{}, nil
	}
	if windowDays <= 0 {
		windowDays = DefaultPriceWindowDays
	}

	all, err := s.products.List(ctx, repository.ProductListFilter{})
	if err != nil {
		return MarketPrice{}, err
	}

	matches := make([]domain.Product, 0, 4)
	for _, candidate := range all {
		if candidate.Status != ProductStatusPublished {
			continue
		}
		if CatalogKey(candidate.Name, candidate.Category) != key {
			continue
		}
		matches = append(matches, candidate)
	}
	if len(matches) == 0 {
		return MarketPrice{}, nil
	}

	market := MarketPrice{CatalogKey: key, ShopCount: countShops(matches)}
	market.CurrentAverage, market.CurrentLowest, market.CurrentHighest = spread(matches)

	if !market.Comparable() {
		// Nothing to average against; the caller hides the comparison.
		return market, nil
	}

	series := make([]marketSeries, 0, len(matches))
	for _, match := range matches {
		points, err := s.priceSeries(ctx, match)
		if err != nil {
			return MarketPrice{}, err
		}
		if len(points) > 0 {
			series = append(series, marketSeries{shopID: match.ShopID, points: points})
		}
	}

	market.History = averageSeries(series, s.now(), windowDays)
	return market, nil
}

// priceSeries rebuilds one product's price timeline from its recorded history,
// falling back to the price it carries now when nothing was ever recorded.
func (s *Service) priceSeries(ctx context.Context, product domain.Product) ([]PricePoint, error) {
	if s.events == nil {
		return []PricePoint{{At: product.CreatedAt, Price: product.Price}}, nil
	}

	events, err := s.events.List(ctx, repository.EventLogListFilter{
		ResourceType: "product",
		ResourceID:   product.ProductID,
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })

	points := make([]PricePoint, 0, len(events))
	for _, event := range events {
		snapshot := decodeMutation(event.PayloadJSON)
		if snapshot.After == nil {
			continue
		}
		points = append(points, PricePoint{
			At:    parseOccurredAt(event),
			Price: snapshot.After.Price,
		})
	}
	if len(points) == 0 {
		points = append(points, PricePoint{At: product.CreatedAt, Price: product.Price})
	}
	return points, nil
}

// averageSeries samples every shop's timeline at each moment any of them
// changed, and averages the prices in effect then.
//
// Sampling on the union of change instants rather than on a fixed grid keeps
// every step a real event: a bend in the line always corresponds to a shop
// actually changing its price.
func averageSeries(series []marketSeries, now time.Time, windowDays int) []PricePoint {
	if len(series) == 0 {
		return nil
	}
	start := now.AddDate(0, 0, -windowDays)

	instants := collectInstants(series, start)
	if len(instants) == 0 {
		return nil
	}

	averaged := make([]PricePoint, 0, len(instants))
	for _, instant := range instants {
		var total float64
		var counted int
		for _, shop := range series {
			// A shop that had not listed the product yet is left out rather
			// than counted as zero, which would drag the average down.
			if price, ok := priceAt(shop.points, instant); ok {
				total += price
				counted++
			}
		}
		if counted == 0 {
			continue
		}
		averaged = append(averaged, PricePoint{At: instant, Price: total / float64(counted)})
	}
	return averaged
}

// collectInstants is every moment any shop changed a price, clamped into the
// window, with the window's start included so the line begins at its edge.
func collectInstants(series []marketSeries, start time.Time) []time.Time {
	seen := map[int64]bool{}
	instants := make([]time.Time, 0)

	add := func(at time.Time) {
		key := at.UnixNano()
		if seen[key] {
			return
		}
		seen[key] = true
		instants = append(instants, at)
	}

	var anyBefore bool
	for _, shop := range series {
		for _, point := range shop.points {
			if point.At.Before(start) {
				anyBefore = true
				continue
			}
			add(point.At)
		}
	}
	if anyBefore {
		add(start)
	}

	sort.Slice(instants, func(i, j int) bool { return instants[i].Before(instants[j]) })
	return instants
}

// priceAt is the price in effect at a moment: the last change at or before it.
// Not ok when the product did not exist yet.
func priceAt(points []PricePoint, instant time.Time) (float64, bool) {
	var price float64
	var found bool
	for _, point := range points {
		if point.At.After(instant) {
			break
		}
		price, found = point.Price, true
	}
	return price, found
}

func countShops(products []domain.Product) int {
	seen := map[string]bool{}
	for _, product := range products {
		seen[product.ShopID] = true
	}
	return len(seen)
}

// spread is the average, lowest and highest price being charged right now.
func spread(products []domain.Product) (average, lowest, highest float64) {
	if len(products) == 0 {
		return 0, 0, 0
	}
	lowest, highest = products[0].Price, products[0].Price
	var total float64
	for _, product := range products {
		total += product.Price
		if product.Price < lowest {
			lowest = product.Price
		}
		if product.Price > highest {
			highest = product.Price
		}
	}
	return total / float64(len(products)), lowest, highest
}
