package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductRepository struct{ collection *mongo.Collection }

func NewProductRepository(db *mongo.Database) *ProductRepository {
	return &ProductRepository{collection: db.Collection(productsCollection)}
}
func (r *ProductRepository) Save(ctx context.Context, product domain.Product) error {
	return saveByID(ctx, r.collection, product.ProductID, product)
}
func (r *ProductRepository) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	return getByID[domain.Product](ctx, r.collection, productID)
}
func (r *ProductRepository) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	query := bson.M{}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.OwnerUserID != "" {
		query["ownerUserId"] = filter.OwnerUserID
	}
	return listDocuments[domain.Product](ctx, r.collection, query)
}
