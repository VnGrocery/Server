package firestore

import (
	"context"
	"fmt"
	"sort"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
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
