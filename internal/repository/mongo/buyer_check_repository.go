package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type BuyerCheckRepository struct{ collection *mongo.Collection }

func NewBuyerCheckRepository(db *mongo.Database) *BuyerCheckRepository {
	return &BuyerCheckRepository{collection: db.Collection(buyerChecksCollection)}
}
func (r *BuyerCheckRepository) Save(ctx context.Context, check domain.BuyerCheck) error {
	return saveByID(ctx, r.collection, check.CheckID, check)
}
func (r *BuyerCheckRepository) GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error) {
	return getByID[domain.BuyerCheck](ctx, r.collection, checkID)
}
func (r *BuyerCheckRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
	return r.listSorted(ctx, bson.M{"shopId": shopID})
}
func (r *BuyerCheckRepository) ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
	return r.listSorted(ctx, bson.M{"buyerUserId": buyerUserID})
}
func (r *BuyerCheckRepository) listSorted(ctx context.Context, filter bson.M) ([]domain.BuyerCheck, error) {
	items, err := listDocuments[domain.BuyerCheck](ctx, r.collection, filter)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
