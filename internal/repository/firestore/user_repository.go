package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
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
		return fmt.Errorf("khong luu duoc user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	doc, err := r.client.Collection(UsersCollection).Doc(userID).Get(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("khong lay duoc user: %w", err)
	}

	var user domain.User
	if err := doc.DataTo(&user); err != nil {
		return domain.User{}, fmt.Errorf("khong map duoc user: %w", err)
	}

	return user, nil
}
