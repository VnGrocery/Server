package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
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
func (r *BuyerCheckRepository) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	query := bson.M{}
	if filter.CheckID != "" {
		query["checkId"] = filter.CheckID
	}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.ProductID != "" {
		query["productId"] = filter.ProductID
	}
	if filter.BuyerUserID != "" {
		query["buyerUserId"] = filter.BuyerUserID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Verdict != "" {
		query["verdict"] = filter.Verdict
	}
	if !filter.CreatedAfter.IsZero() || !filter.CreatedBefore.IsZero() {
		rangeFilter := bson.M{}
		if !filter.CreatedAfter.IsZero() {
			rangeFilter["$gte"] = filter.CreatedAfter
		}
		if !filter.CreatedBefore.IsZero() {
			rangeFilter["$lte"] = filter.CreatedBefore
		}
		query["createdAt"] = rangeFilter
	}
	return r.listSorted(ctx, query)
}
func (r *BuyerCheckRepository) listSorted(ctx context.Context, filter bson.M) ([]domain.BuyerCheck, error) {
	items, err := listDocuments[domain.BuyerCheck](ctx, r.collection, filter)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
