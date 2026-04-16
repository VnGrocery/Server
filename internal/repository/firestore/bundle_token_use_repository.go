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
