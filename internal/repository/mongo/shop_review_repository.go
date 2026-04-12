package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type ShopReviewRepository struct{ collection *mongo.Collection }

func NewShopReviewRepository(db *mongo.Database) *ShopReviewRepository {
	return &ShopReviewRepository{collection: db.Collection(shopReviewsCollection)}
}
func (r *ShopReviewRepository) Save(ctx context.Context, review domain.ShopReview) error {
	return saveByID(ctx, r.collection, review.ReviewID, review)
}
func (r *ShopReviewRepository) GetByShopAndUser(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error) {
	items, err := listDocuments[domain.ShopReview](ctx, r.collection, bson.M{"shopId": shopID, "reviewerUserId": reviewerUserID})
	if err != nil {
		return domain.ShopReview{}, err
	}
	if len(items) == 0 {
		return domain.ShopReview{}, nil
	}
	return items[0], nil
}
func (r *ShopReviewRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	items, err := listDocuments[domain.ShopReview](ctx, r.collection, bson.M{"shopId": shopID})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}
