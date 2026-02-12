package firestore

import (
	"context"
	"fmt"
	"sort"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type PledgeRepository struct {
	client *gofirestore.Client
}

func NewPledgeRepository(client *gofirestore.Client) *PledgeRepository {
	return &PledgeRepository{client: client}
}

func (r *PledgeRepository) Save(ctx context.Context, pledge domain.Pledge) error {
	_, err := r.client.Collection(PledgesCollection).Doc(pledge.PledgeID).Set(ctx, pledge)
	if err != nil {
		return fmt.Errorf("failed to save pledge: %w", err)
	}

	return nil
}

func (r *PledgeRepository) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	doc, err := r.client.Collection(PledgesCollection).Doc(pledgeID).Get(ctx)
	if err != nil {
		return domain.Pledge{}, fmt.Errorf("failed to get pledge: %w", err)
	}

	var pledge domain.Pledge
	if err := doc.DataTo(&pledge); err != nil {
		return domain.Pledge{}, fmt.Errorf("failed to decode pledge document: %w", err)
	}

	return pledge, nil
}

func (r *PledgeRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	docs, err := r.client.Collection(PledgesCollection).Where("shopId", "==", shopID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list pledges by shop: %w", err)
	}

	pledges := make([]domain.Pledge, 0, len(docs))
	for _, doc := range docs {
		var pledge domain.Pledge
		if err := doc.DataTo(&pledge); err != nil {
			return nil, fmt.Errorf("failed to decode pledge document: %w", err)
		}
		pledges = append(pledges, pledge)
	}

	sort.Slice(pledges, func(i, j int) bool {
		return pledges[i].CreatedAt.After(pledges[j].CreatedAt)
	})

	return pledges, nil
}
