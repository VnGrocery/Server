package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductBatchRepository struct {
	client *gofirestore.Client
}

func NewProductBatchRepository(client *gofirestore.Client) *ProductBatchRepository {
	return &ProductBatchRepository{client: client}
}

func (r *ProductBatchRepository) Save(ctx context.Context, batch domain.ProductBatch) error {
	_, err := r.client.Collection(ProductBatchesCollection).Doc(batch.BatchID).Set(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to save product batch: %w", err)
	}
	return nil
}

func (r *ProductBatchRepository) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	doc, err := r.client.Collection(ProductBatchesCollection).Doc(batchID).Get(ctx)
	if err != nil {
		return domain.ProductBatch{}, fmt.Errorf("failed to get product batch: %w", err)
	}

	var batch domain.ProductBatch
	if err := doc.DataTo(&batch); err != nil {
		return domain.ProductBatch{}, fmt.Errorf("failed to decode product batch document: %w", err)
	}
	return batch, nil
}

func (r *ProductBatchRepository) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	query := r.client.Collection(ProductBatchesCollection).Query
	if filter.ShopID != "" {
		query = query.Where("shopId", "==", filter.ShopID)
	}
	if filter.ProductID != "" {
		query = query.Where("productId", "==", filter.ProductID)
	}
	if filter.Status != "" {
		query = query.Where("status", "==", filter.Status)
	}
	if filter.OwnerUserID != "" {
		query = query.Where("ownerUserId", "==", filter.OwnerUserID)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list product batches: %w", err)
	}

	batches := make([]domain.ProductBatch, 0, len(docs))
	for _, doc := range docs {
		var batch domain.ProductBatch
		if err := doc.DataTo(&batch); err != nil {
			return nil, fmt.Errorf("failed to decode product batch document: %w", err)
		}
		batches = append(batches, batch)
	}
	return batches, nil
}
