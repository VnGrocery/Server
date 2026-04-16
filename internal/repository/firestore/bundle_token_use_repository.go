package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vngrocery/internal/domain"
)

type BundleTokenUseRepository struct {
	client *gofirestore.Client
}

func NewBundleTokenUseRepository(client *gofirestore.Client) *BundleTokenUseRepository {
	return &BundleTokenUseRepository{client: client}
}

func (r *BundleTokenUseRepository) Reserve(ctx context.Context, usage domain.BundleTokenUse) (bool, error) {
	_, err := r.client.Collection(BundleTokenUsesCollection).Doc(usage.UseID).Create(ctx, usage)
	if err == nil {
		return true, nil
	}
	if status.Code(err) == codes.AlreadyExists {
		return false, nil
	}
	return false, fmt.Errorf("failed to reserve bundle token nonce: %w", err)
}

func (r *BundleTokenUseRepository) GetByID(ctx context.Context, useID string) (domain.BundleTokenUse, error) {
	doc, err := r.client.Collection(BundleTokenUsesCollection).Doc(useID).Get(ctx)
	if err != nil {
		return domain.BundleTokenUse{}, fmt.Errorf("failed to get bundle token use: %w", err)
	}
	var usage domain.BundleTokenUse
	if err := doc.DataTo(&usage); err != nil {
		return domain.BundleTokenUse{}, fmt.Errorf("failed to decode bundle token use: %w", err)
	}
	return usage, nil
}
