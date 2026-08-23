package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductCommentRepository struct{ collection *mongo.Collection }

func NewProductCommentRepository(db *mongo.Database) *ProductCommentRepository {
	return &ProductCommentRepository{collection: db.Collection(productCommentsCollection)}
}
func (r *ProductCommentRepository) Save(ctx context.Context, comment domain.ProductComment) error {
	return saveByID(ctx, r.collection, comment.CommentID, comment)
}
func (r *ProductCommentRepository) GetByID(ctx context.Context, commentID string) (domain.ProductComment, error) {
	return getByID[domain.ProductComment](ctx, r.collection, commentID)
}
func (r *ProductCommentRepository) List(ctx context.Context, filter repository.ProductCommentListFilter) ([]domain.ProductComment, error) {
	query := bson.M{}
	if filter.CommentID != "" {
		query["commentId"] = filter.CommentID
	}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.ProductID != "" {
		query["productId"] = filter.ProductID
	}
	if filter.AuthorUserID != "" {
		query["authorUserId"] = filter.AuthorUserID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	items, err := listDocuments[domain.ProductComment](ctx, r.collection, query)
	if err != nil {
		return nil, err
	}
	// Newest first: a stall's goods change daily, so the comment written this
	// morning matters more than the one from last month.
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
