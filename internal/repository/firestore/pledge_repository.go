package firestore

import (
	"context"
	"fmt"

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
