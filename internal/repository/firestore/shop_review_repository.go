package firestore

import (
	"context"
	"fmt"
	"sort"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type ShopReviewRepository struct {
	client *gofirestore.Client
}

func NewShopReviewRepository(client *gofirestore.Client) *ShopReviewRepository {
	return &ShopReviewRepository{client: client}
}

func (r *ShopReviewRepository) Save(ctx context.Context, review domain.ShopReview) error {
	_, err := r.client.Collection(ShopReviewsCollection).Doc(review.ReviewID).Set(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to save shop review: %w", err)
	}
	return nil
}

func (r *ShopReviewRepository) GetByShopAndUser(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error) {
	docs, err := r.client.Collection(ShopReviewsCollection).
		Where("shopId", "==", shopID).
		Where("reviewerUserId", "==", reviewerUserID).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return domain.ShopReview{}, fmt.Errorf("failed to get shop review: %w", err)
	}
	if len(docs) == 0 {
		return domain.ShopReview{}, nil
	}

	var review domain.ShopReview
	if err := docs[0].DataTo(&review); err != nil {
		return domain.ShopReview{}, fmt.Errorf("failed to decode shop review document: %w", err)
	}
	return review, nil
}

func (r *ShopReviewRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	docs, err := r.client.Collection(ShopReviewsCollection).Where("shopId", "==", shopID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list shop reviews: %w", err)
	}

	reviews := make([]domain.ShopReview, 0, len(docs))
	for _, doc := range docs {
		var review domain.ShopReview
		if err := doc.DataTo(&review); err != nil {
			return nil, fmt.Errorf("failed to decode shop review document: %w", err)
		}
		reviews = append(reviews, review)
	}

	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].UpdatedAt.After(reviews[j].UpdatedAt)
	})

	return reviews, nil
}
