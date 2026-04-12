package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type PasswordResetTokenRepository struct{ collection *mongo.Collection }

func NewPasswordResetTokenRepository(db *mongo.Database) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{collection: db.Collection(passwordResetTokensCollection)}
}
func (r *PasswordResetTokenRepository) Save(ctx context.Context, token domain.PasswordResetToken) error {
	return saveByID(ctx, r.collection, token.TokenID, token)
}
func (r *PasswordResetTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error) {
	var doc bson.M
	if err := r.collection.FindOne(ctx, bson.M{"tokenHash": strings.TrimSpace(tokenHash)}).Decode(&doc); err != nil {
		return domain.PasswordResetToken{}, fmt.Errorf("query password reset token: %w", err)
	}
	var token domain.PasswordResetToken
	if err := decodeDocument(doc, &token); err != nil {
		return domain.PasswordResetToken{}, err
	}
	return token, nil
}
