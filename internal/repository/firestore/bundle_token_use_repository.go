package firestore

import (
	"context"
	"fmt"
	"time"

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

func (r *BundleTokenUseRepository) DeleteExpired(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	docs, err := r.client.Collection(BundleTokenUsesCollection).
		Where("expiresAt", "<", before).
		Limit(limit).
		Documents(ctx).
		GetAll()
	if err != nil {
		return 0, fmt.Errorf("failed to list expired bundle token uses: %w", err)
	}
	if len(docs) == 0 {
		return 0, nil
	}
	batch := r.client.Batch()
	for _, doc := range docs {
		batch.Delete(doc.Ref)
	}
	_, err = batch.Commit(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired bundle token uses: %w", err)
	}
	return len(docs), nil
}
