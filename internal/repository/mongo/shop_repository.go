package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ShopRepository struct{ collection *mongo.Collection }

func NewShopRepository(db *mongo.Database) *ShopRepository {
	return &ShopRepository{collection: db.Collection(shopsCollection)}
}
func (r *ShopRepository) Save(ctx context.Context, shop domain.Shop) error {
	return saveByID(ctx, r.collection, shop.ShopID, shop)
}
func (r *ShopRepository) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return getByID[domain.Shop](ctx, r.collection, shopID)
}
func (r *ShopRepository) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	query := bson.M{}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.OwnerUserID != "" {
		query["ownerUserId"] = filter.OwnerUserID
	}
	return listDocuments[domain.Shop](ctx, r.collection, query)
}
