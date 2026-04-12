package mongo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type AuthUserRepository struct{ collection *mongo.Collection }

func NewAuthUserRepository(db *mongo.Database) *AuthUserRepository {
	return &AuthUserRepository{collection: db.Collection(authUsersCollection)}
}
func (r *AuthUserRepository) NewUserID() string { return uuid.NewString() }
func (r *AuthUserRepository) Save(ctx context.Context, user domain.AuthUser) error {
	return saveByID(ctx, r.collection, user.UserID, user)
}
func (r *AuthUserRepository) GetByID(ctx context.Context, userID string) (domain.AuthUser, error) {
	return getByID[domain.AuthUser](ctx, r.collection, strings.TrimSpace(userID))
}
func (r *AuthUserRepository) GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error) {
	return r.findOne(ctx, bson.M{"emailLower": strings.ToLower(strings.TrimSpace(emailLower))})
}
func (r *AuthUserRepository) GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error) {
	return r.findOne(ctx, bson.M{"googleSub": strings.TrimSpace(googleSub)})
}
func (r *AuthUserRepository) findOne(ctx context.Context, filter bson.M) (domain.AuthUser, error) {
	var doc bson.M
	if err := r.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		return domain.AuthUser{}, fmt.Errorf("find auth user: %w", err)
	}
	var user domain.AuthUser
	if err := decodeDocument(doc, &user); err != nil {
		return domain.AuthUser{}, err
	}
	return user, nil
}
