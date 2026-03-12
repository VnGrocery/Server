package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type UserRepository struct {
	client *gofirestore.Client
}

func NewUserRepository(client *gofirestore.Client) *UserRepository {
	return &UserRepository{client: client}
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	_, err := r.client.Collection(UsersCollection).Doc(user.UserID).Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	doc, err := r.client.Collection(UsersCollection).Doc(userID).Get(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to get user: %w", err)
	}

	var user domain.User
	if err := doc.DataTo(&user); err != nil {
		return domain.User{}, fmt.Errorf("failed to decode user document: %w", err)
	}

	return user, nil
}

func (r *UserRepository) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	query := r.client.Collection(UsersCollection).Query
	if filter.Status != "" {
		query = query.Where("status", "==", filter.Status)
	}
	if filter.Role != "" {
		query = query.Where("role", "==", filter.Role)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]domain.User, 0, len(docs))
	for _, doc := range docs {
		var user domain.User
		if err := doc.DataTo(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user document: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}
