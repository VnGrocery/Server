package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type RefreshTokenRepository struct{ collection *mongo.Collection }

func NewRefreshTokenRepository(db *mongo.Database) *RefreshTokenRepository {
	return &RefreshTokenRepository{collection: db.Collection(refreshTokensCollection)}
}
func (r *RefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	return saveByID(ctx, r.collection, token.TokenID, token)
}
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var doc bson.M
	if err := r.collection.FindOne(ctx, bson.M{"tokenHash": strings.TrimSpace(tokenHash)}).Decode(&doc); err != nil {
		return domain.RefreshToken{}, fmt.Errorf("query refresh token: %w", err)
	}
	var token domain.RefreshToken
	if err := decodeDocument(doc, &token); err != nil {
		return domain.RefreshToken{}, err
	}
	return token, nil
}
