package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"vngrocery/internal/domain"
)

type BundleTokenUseRepository struct {
	collection *mongo.Collection
}

func NewBundleTokenUseRepository(db *mongo.Database) *BundleTokenUseRepository {
	return &BundleTokenUseRepository{collection: db.Collection(bundleTokenUsesCollection)}
}

func (r *BundleTokenUseRepository) Reserve(ctx context.Context, usage domain.BundleTokenUse) (bool, error) {
	doc, err := encodeDocument(usage)
	if err != nil {
		return false, err
	}
	doc["_id"] = usage.UseID

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": usage.UseID},
		bson.M{"$setOnInsert": doc},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return false, fmt.Errorf("reserve %s: %w", r.collection.Name(), err)
	}
	return result.UpsertedCount > 0, nil
}

func (r *BundleTokenUseRepository) GetByID(ctx context.Context, useID string) (domain.BundleTokenUse, error) {
	return getByID[domain.BundleTokenUse](ctx, r.collection, useID)
}
