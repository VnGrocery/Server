package firestore

import (
	"context"
	"fmt"
	"strings"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type PasswordResetTokenRepository struct {
	client *gofirestore.Client
}

func NewPasswordResetTokenRepository(client *gofirestore.Client) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{client: client}
}

func (r *PasswordResetTokenRepository) Save(ctx context.Context, token domain.PasswordResetToken) error {
	_, err := r.client.Collection(PasswordResetTokensCollection).Doc(token.TokenID).Set(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to save password reset token: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return domain.PasswordResetToken{}, fmt.Errorf("tokenHash is required")
	}

	docs, err := r.client.Collection(PasswordResetTokensCollection).
		Where("tokenHash", "==", tokenHash).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return domain.PasswordResetToken{}, fmt.Errorf("failed to query password reset token: %w", err)
	}
	if len(docs) == 0 {
		return domain.PasswordResetToken{}, fmt.Errorf("password reset token not found")
	}

	var token domain.PasswordResetToken
	if err := docs[0].DataTo(&token); err != nil {
		return domain.PasswordResetToken{}, fmt.Errorf("failed to decode password reset token: %w", err)
	}
	return token, nil
}
