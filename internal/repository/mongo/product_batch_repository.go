package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductBatchRepository struct{ collection *mongo.Collection }

func NewProductBatchRepository(db *mongo.Database) *ProductBatchRepository {
	return &ProductBatchRepository{collection: db.Collection(productBatchesCollection)}
}

func (r *ProductBatchRepository) Save(ctx context.Context, batch domain.ProductBatch) error {
	return saveByID(ctx, r.collection, batch.BatchID, batch)
}

func (r *ProductBatchRepository) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	return getByID[domain.ProductBatch](ctx, r.collection, batchID)
}

func (r *ProductBatchRepository) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	query := bson.M{}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.ProductID != "" {
		query["productId"] = filter.ProductID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.OwnerUserID != "" {
		query["ownerUserId"] = filter.OwnerUserID
	}
	return listDocuments[domain.ProductBatch](ctx, r.collection, query)
}
