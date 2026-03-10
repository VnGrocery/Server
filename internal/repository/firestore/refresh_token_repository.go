package firestore

import (
	"context"
	"fmt"
	"strings"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type RefreshTokenRepository struct {
	client *gofirestore.Client
}

func NewRefreshTokenRepository(client *gofirestore.Client) *RefreshTokenRepository {
	return &RefreshTokenRepository{client: client}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	_, err := r.client.Collection(RefreshTokensCollection).Doc(token.TokenID).Set(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return domain.RefreshToken{}, fmt.Errorf("tokenHash is required")
	}

	docs, err := r.client.Collection(RefreshTokensCollection).
		Where("tokenHash", "==", tokenHash).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("failed to query refresh token: %w", err)
	}
	if len(docs) == 0 {
		return domain.RefreshToken{}, fmt.Errorf("refresh token not found")
	}

	var token domain.RefreshToken
	if err := docs[0].DataTo(&token); err != nil {
		return domain.RefreshToken{}, fmt.Errorf("failed to decode refresh token: %w", err)
	}
	return token, nil
}
