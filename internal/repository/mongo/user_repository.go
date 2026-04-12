package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type UserRepository struct{ collection *mongo.Collection }

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collection: db.Collection(usersCollection)}
}
func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	return saveByID(ctx, r.collection, user.UserID, user)
}
func (r *UserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	return getByID[domain.User](ctx, r.collection, userID)
}
func (r *UserRepository) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	query := bson.M{}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Role != "" {
		query["role"] = filter.Role
	}
	return listDocuments[domain.User](ctx, r.collection, query)
}
