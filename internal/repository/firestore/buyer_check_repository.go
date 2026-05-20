package firestore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type BuyerCheckRepository struct {
	client *gofirestore.Client
}

func NewBuyerCheckRepository(client *gofirestore.Client) *BuyerCheckRepository {
	return &BuyerCheckRepository{client: client}
}

func (r *BuyerCheckRepository) Save(ctx context.Context, check domain.BuyerCheck) error {
	_, err := r.client.Collection(BuyerChecksCollection).Doc(check.CheckID).Set(ctx, check)
	if err != nil {
		return fmt.Errorf("failed to save buyer check: %w", err)
	}

	return nil
}

func (r *BuyerCheckRepository) GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error) {
	doc, err := r.client.Collection(BuyerChecksCollection).Doc(checkID).Get(ctx)
	if err != nil {
		return domain.BuyerCheck{}, fmt.Errorf("failed to get buyer check: %w", err)
	}

	var check domain.BuyerCheck
	if err := doc.DataTo(&check); err != nil {
		return domain.BuyerCheck{}, fmt.Errorf("failed to decode buyer check document: %w", err)
	}
	return check, nil
}

func (r *BuyerCheckRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
	docs, err := r.client.Collection(BuyerChecksCollection).Where("shopId", "==", shopID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list buyer checks by shop: %w", err)
	}

	return decodeBuyerChecks(docs)
}

func (r *BuyerCheckRepository) ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
	docs, err := r.client.Collection(BuyerChecksCollection).Where("buyerUserId", "==", buyerUserID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list buyer checks by user: %w", err)
	}

	return decodeBuyerChecks(docs)
}

func (r *BuyerCheckRepository) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	docs, err := r.client.Collection(BuyerChecksCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list buyer checks: %w", err)
	}
	checks, err := decodeBuyerChecks(docs)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.BuyerCheck, 0, len(checks))
	for _, check := range checks {
		if strings.TrimSpace(filter.CheckID) != "" && check.CheckID != strings.TrimSpace(filter.CheckID) {
			continue
		}
		if strings.TrimSpace(filter.ShopID) != "" && check.ShopID != strings.TrimSpace(filter.ShopID) {
			continue
		}
		if strings.TrimSpace(filter.BundleID) != "" && check.BundleID != strings.TrimSpace(filter.BundleID) {
			continue
		}
		if strings.TrimSpace(filter.ProductID) != "" && check.ProductID != strings.TrimSpace(filter.ProductID) {
			continue
		}
		if strings.TrimSpace(filter.BatchID) != "" && check.BatchID != strings.TrimSpace(filter.BatchID) {
			continue
		}
		if strings.TrimSpace(filter.BuyerUserID) != "" && check.BuyerUserID != strings.TrimSpace(filter.BuyerUserID) {
			continue
		}
		if strings.TrimSpace(filter.Status) != "" && check.Status != strings.TrimSpace(filter.Status) {
			continue
		}
		if strings.TrimSpace(filter.Verdict) != "" && check.Verdict != strings.TrimSpace(filter.Verdict) {
			continue
		}
		if strings.TrimSpace(filter.LocationStatus) != "" && check.LocationStatus != strings.TrimSpace(filter.LocationStatus) {
			continue
		}
		if !filter.CreatedAfter.IsZero() && check.CreatedAt.Before(filter.CreatedAfter) {
			continue
		}
		if !filter.CreatedBefore.IsZero() && check.CreatedAt.After(filter.CreatedBefore) {
			continue
		}
		filtered = append(filtered, check)
	}
	return filtered, nil
}

func decodeBuyerChecks(docs []*gofirestore.DocumentSnapshot) ([]domain.BuyerCheck, error) {
	checks := make([]domain.BuyerCheck, 0, len(docs))
	for _, doc := range docs {
		var check domain.BuyerCheck
		if err := doc.DataTo(&check); err != nil {
			return nil, fmt.Errorf("failed to decode buyer check document: %w", err)
		}
		checks = append(checks, check)
	}

	sort.Slice(checks, func(i, j int) bool {
		return checks[i].CreatedAt.After(checks[j].CreatedAt)
	})

	return checks, nil
}
