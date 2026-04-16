package mongo

import (
	"context"
	"fmt"
	"time"

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

func (r *BundleTokenUseRepository) DeleteExpired(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	cursor, err := r.collection.Find(
		ctx,
		bson.M{"expiresAt": bson.M{"$lt": before}},
		options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(int64(limit)),
	)
	if err != nil {
		return 0, fmt.Errorf("list expired %s: %w", r.collection.Name(), err)
	}
	defer cursor.Close(ctx)

	ids := make([]string, 0, limit)
	for cursor.Next(ctx) {
		var row struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&row); err != nil {
			return 0, fmt.Errorf("decode expired %s: %w", r.collection.Name(), err)
		}
		if row.ID != "" {
			ids = append(ids, row.ID)
		}
	}
	if err := cursor.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired %s: %w", r.collection.Name(), err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.collection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return 0, fmt.Errorf("delete expired %s: %w", r.collection.Name(), err)
	}
	return int(result.DeletedCount), nil
}
