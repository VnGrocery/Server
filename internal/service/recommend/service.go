package recommend

import (
	"context"
	"fmt"
	"strings"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	shopsvc "vngrocery/internal/service/shop"
)

// How much each kind of engagement counts towards a category.
//
// Checking a product is the strongest: the reader stood in front of it and
// photographed it. Reviewing a shop is next. Saving a voucher is the weakest --
// it says something, but a saved voucher is not a visit.
const (
	weightCheck   = 3.0
	weightReview  = 2.0
	weightVoucher = 1.0
)

// Shops is the slice of the shop service this needs.
type Shops interface {
	List(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error)
}

// Service builds suggestions from what a reader has done.
type Service struct {
	shops     Shops
	products  repository.ProductRepository
	reviews   repository.ShopReviewRepository
	checks    repository.BuyerCheckRepository
	userVouch repository.UserVoucherRepository
	vouchers  repository.VoucherRepository
}

func NewService(
	shops Shops,
	products repository.ProductRepository,
	reviews repository.ShopReviewRepository,
	checks repository.BuyerCheckRepository,
	userVouchers repository.UserVoucherRepository,
	vouchers repository.VoucherRepository,
) *Service {
	return &Service{
		shops:     shops,
		products:  products,
		reviews:   reviews,
		checks:    checks,
		userVouch: userVouchers,
		vouchers:  vouchers,
	}
}

// Input asks for one reader's suggestions.
type Input struct {
	UserID string

	// Near narrows and orders by distance when the reader shared a location.
	Near shopsvc.NearFilter

	Limit int
}

// Result is the suggestions and an honest account of what they rest on.
type Result struct {
	Products []ProductSuggestion
	Shops    []ShopSuggestion

	// Personalised is false when the reader has done nothing the app can learn
	// from. The list is then ordered by trust and distance, and the client must
	// not present it as personal.
	Personalised bool

	// SignalCount and Categories say what the suggestions were derived from, so
	// the reader can see why they are being shown these.
	SignalCount int
	Categories  []string
}

type ProductSuggestion struct {
	Product    domain.Product
	Shop       domain.Shop
	Score      float64
	Reasons    []string
	DistanceKm *float64
}

type ShopSuggestion struct {
	Shop       domain.Shop
	Trust      shopsvc.TrustSummary
	Rating     shopsvc.RatingSummary
	Score      float64
	Reasons    []string
	DistanceKm *float64
}

const defaultLimit = 10

// Suggest returns shops and products for one reader.
func (s *Service) Suggest(ctx context.Context, input Input) (Result, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return Result{}, fmt.Errorf("userId is required")
	}
	if s.shops == nil || s.products == nil {
		return Result{}, fmt.Errorf("recommendation dependencies are not configured")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	signals, err := s.signalsFor(ctx, userID)
	if err != nil {
		return Result{}, err
	}

	// The same listing the discovery screens use, so a suggestion can never
	// point at a shop those screens would have hidden.
	listed, err := s.shops.List(ctx, shopsvc.ListInput{
		Page:     1,
		PageSize: 200,
		Near:     input.Near,
	})
	if err != nil {
		return Result{}, err
	}

	shopByID := make(map[string]shopsvc.ShopView, len(listed.Items))
	shopCandidates := make([]Candidate, 0, len(listed.Items))
	for _, view := range listed.Items {
		shopByID[view.Shop.ShopID] = view
		shopCandidates = append(shopCandidates, Candidate{
			ID:         view.Shop.ShopID,
			ShopID:     view.Shop.ShopID,
			Category:   dominantCategory(ctx, s.products, view.Shop.ShopID),
			TrustScore: view.TrustSummary.Score,
			Rating:     view.RatingSummary.AverageRating,
			DistanceKm: view.DistanceKm,
		})
	}

	result := Result{
		Personalised: signals.HasSignal(),
		SignalCount:  signals.Count,
		Categories:   topCategories(signals),
	}

	for _, suggestion := range Rank(shopCandidates, signals, limit) {
		view := shopByID[suggestion.Candidate.ShopID]
		result.Shops = append(result.Shops, ShopSuggestion{
			Shop:       view.Shop,
			Trust:      view.TrustSummary,
			Rating:     view.RatingSummary,
			Score:      suggestion.Score,
			Reasons:    suggestion.Reasons,
			DistanceKm: view.DistanceKm,
		})
	}

	productCandidates, productByID := s.productCandidates(ctx, shopByID)
	for _, suggestion := range Rank(productCandidates, signals, limit) {
		product := productByID[suggestion.Candidate.ID]
		view := shopByID[product.ShopID]
		result.Products = append(result.Products, ProductSuggestion{
			Product:    product,
			Shop:       view.Shop,
			Score:      suggestion.Score,
			Reasons:    suggestion.Reasons,
			DistanceKm: view.DistanceKm,
		})
	}

	return result, nil
}

// signalsFor gathers everything the reader has done. A repository that cannot
// be reached costs its own signal and nothing else: a partial picture still
// beats refusing to suggest anything.
func (s *Service) signalsFor(ctx context.Context, userID string) (Signals, error) {
	signals := Signals{
		CategoryWeight: map[string]float64{},
		ShopsRated:     map[string]bool{},
	}

	if s.reviews != nil {
		reviews, err := s.reviews.ListByReviewerUserID(ctx, userID)
		if err != nil {
			return Signals{}, err
		}
		for _, review := range reviews {
			signals.ShopsRated[review.ShopID] = true
			signals.Count++
			// A review is about a shop, so the categories it speaks for are the
			// ones that shop sells.
			for _, category := range shopCategories(ctx, s.products, review.ShopID) {
				signals.CategoryWeight[category] += weightReview
			}
		}
	}

	if s.checks != nil {
		checks, err := s.checks.ListByBuyerUserID(ctx, userID)
		if err != nil {
			return Signals{}, err
		}
		for _, check := range checks {
			signals.Count++
			// The pledged category is what the seller declared; the actual one
			// is what the photo showed. Either identifies the product.
			category := strings.TrimSpace(check.ActualCategory)
			if category == "" {
				category = strings.TrimSpace(check.PledgedCategory)
			}
			if category != "" {
				signals.CategoryWeight[category] += weightCheck
			}
		}
	}

	if s.userVouch != nil && s.vouchers != nil {
		saved, err := s.userVouch.ListByUserID(ctx, userID)
		if err != nil {
			return Signals{}, err
		}
		for _, item := range saved {
			voucher, err := s.vouchers.GetByID(ctx, item.VoucherID)
			if err != nil || voucher.ShopID == "" {
				continue
			}
			signals.Count++
			for _, category := range shopCategories(ctx, s.products, voucher.ShopID) {
				signals.CategoryWeight[category] += weightVoucher
			}
		}
	}

	return signals, nil
}

func (s *Service) productCandidates(
	ctx context.Context,
	shops map[string]shopsvc.ShopView,
) ([]Candidate, map[string]domain.Product) {
	candidates := make([]Candidate, 0)
	byID := make(map[string]domain.Product)

	for shopID, view := range shops {
		products, err := s.products.List(ctx, repository.ProductListFilter{ShopID: shopID})
		if err != nil {
			continue
		}
		for _, product := range products {
			if product.Status != publishedStatus {
				continue
			}
			byID[product.ProductID] = product
			candidates = append(candidates, Candidate{
				ID:         product.ProductID,
				ShopID:     shopID,
				Category:   product.Category,
				TrustScore: view.TrustSummary.Score,
				Rating:     view.RatingSummary.AverageRating,
				DistanceKm: view.DistanceKm,
			})
		}
	}
	return candidates, byID
}

const publishedStatus = "published"

// shopCategories is what a shop sells, deduplicated.
func shopCategories(ctx context.Context, products repository.ProductRepository, shopID string) []string {
	if products == nil {
		return nil
	}
	items, err := products.List(ctx, repository.ProductListFilter{ShopID: shopID})
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	categories := make([]string, 0, 4)
	for _, product := range items {
		category := strings.TrimSpace(product.Category)
		if category == "" || seen[category] {
			continue
		}
		seen[category] = true
		categories = append(categories, category)
	}
	return categories
}

// dominantCategory is the one a shop sells most of, used to score the shop as a
// whole against the reader's affinities.
func dominantCategory(ctx context.Context, products repository.ProductRepository, shopID string) string {
	if products == nil {
		return ""
	}
	items, err := products.List(ctx, repository.ProductListFilter{ShopID: shopID})
	if err != nil {
		return ""
	}

	counts := map[string]int{}
	best, bestCount := "", 0
	for _, product := range items {
		category := strings.TrimSpace(product.Category)
		if category == "" {
			continue
		}
		counts[category]++
		// Ties break on the name so the answer does not change between calls.
		if counts[category] > bestCount || (counts[category] == bestCount && category < best) {
			best, bestCount = category, counts[category]
		}
	}
	return best
}

// topCategories names what the reader engages with most, strongest first.
func topCategories(signals Signals) []string {
	type pair struct {
		category string
		weight   float64
	}
	pairs := make([]pair, 0, len(signals.CategoryWeight))
	for category, weight := range signals.CategoryWeight {
		pairs = append(pairs, pair{category, weight})
	}
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0; j-- {
			if pairs[j].weight > pairs[j-1].weight ||
				(pairs[j].weight == pairs[j-1].weight && pairs[j].category < pairs[j-1].category) {
				pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
				continue
			}
			break
		}
	}

	const most = 3
	names := make([]string, 0, most)
	for _, item := range pairs {
		if len(names) == most {
			break
		}
		names = append(names, item.category)
	}
	return names
}
