package firestore

import (
	"context"
	"fmt"
	"strings"

	gofirestore "cloud.google.com/go/firestore"
	"github.com/google/uuid"

	"vngrocery/internal/domain"
)

type AuthUserRepository struct {
	client *gofirestore.Client
}

func NewAuthUserRepository(client *gofirestore.Client) *AuthUserRepository {
	return &AuthUserRepository{client: client}
}

func (r *AuthUserRepository) NewUserID() string {
	return uuid.NewString()
}

func (r *AuthUserRepository) Save(ctx context.Context, user domain.AuthUser) error {
	_, err := r.client.Collection(AuthUsersCollection).Doc(user.UserID).Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save auth user: %w", err)
	}
	return nil
}

func (r *AuthUserRepository) GetByID(ctx context.Context, userID string) (domain.AuthUser, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.AuthUser{}, fmt.Errorf("userID is required")
	}

	doc, err := r.client.Collection(AuthUsersCollection).Doc(userID).Get(ctx)
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to get auth user: %w", err)
	}

	var user domain.AuthUser
	if err := doc.DataTo(&user); err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to decode auth user: %w", err)
	}
	return user, nil
}

func (r *AuthUserRepository) GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error) {
	emailLower = strings.ToLower(strings.TrimSpace(emailLower))
	if emailLower == "" {
		return domain.AuthUser{}, fmt.Errorf("emailLower is required")
	}

	query := r.client.Collection(AuthUsersCollection).Where("emailLower", "==", emailLower).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to query auth user by email: %w", err)
	}
	if len(docs) == 0 {
		return domain.AuthUser{}, fmt.Errorf("auth user not found")
	}

	var user domain.AuthUser
	if err := docs[0].DataTo(&user); err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to decode auth user: %w", err)
	}
	return user, nil
}

func (r *AuthUserRepository) GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error) {
	googleSub = strings.TrimSpace(googleSub)
	if googleSub == "" {
		return domain.AuthUser{}, fmt.Errorf("googleSub is required")
	}

	query := r.client.Collection(AuthUsersCollection).Where("googleSub", "==", googleSub).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to query auth user by google sub: %w", err)
	}
	if len(docs) == 0 {
		return domain.AuthUser{}, fmt.Errorf("auth user not found")
	}

	var user domain.AuthUser
	if err := docs[0].DataTo(&user); err != nil {
		return domain.AuthUser{}, fmt.Errorf("failed to decode auth user: %w", err)
	}
	return user, nil
}
