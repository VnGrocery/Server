package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection names. These were previously re-exported from the Firestore
// repository package; that backend has been removed, so they live here with the
// only backend that still uses them.
const (
	usersCollection                   = "users"
	authUsersCollection               = "auth_users"
	refreshTokensCollection           = "refresh_tokens"
	passwordResetTokensCollection     = "password_reset_tokens"
	bundleTokenUsesCollection         = "bundle_token_uses"
	shopsCollection                   = "shops"
	productsCollection                = "products"
	productFreshnessReportsCollection = "product_freshness_reports"
	pledgesCollection                 = "pledges"
	buyerChecksCollection             = "buyer_checks"
	shopReviewsCollection             = "shop_reviews"
	vouchersCollection                = "vouchers"
	userVouchersCollection            = "user_vouchers"
	eventLogsCollection               = "event_logs"
)

func saveByID(ctx context.Context, collection *mongo.Collection, id string, value any) error {
	doc, err := encodeDocument(value)
	if err != nil {
		return err
	}
	doc["_id"] = id
	_, err = collection.UpdateByID(ctx, id, bson.M{"$set": doc}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save %s: %w", collection.Name(), err)
	}
	return nil
}

func getByID[T any](ctx context.Context, collection *mongo.Collection, id string) (T, error) {
	var zero T
	var doc bson.M
	if err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		return zero, fmt.Errorf("get %s by id: %w", collection.Name(), err)
	}
	var out T
	if err := decodeDocument(doc, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func listDocuments[T any](ctx context.Context, collection *mongo.Collection, filter bson.M) ([]T, error) {
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", collection.Name(), err)
	}
	defer cursor.Close(ctx)

	var items []T
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode %s: %w", collection.Name(), err)
		}
		var item T
		if err := decodeDocument(doc, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", collection.Name(), err)
	}
	return items, nil
}
