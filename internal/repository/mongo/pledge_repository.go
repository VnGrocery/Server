package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"vngrocery/internal/domain"
)

type PledgeRepository struct{ collection *mongo.Collection }

func NewPledgeRepository(db *mongo.Database) *PledgeRepository {
	return &PledgeRepository{collection: db.Collection(pledgesCollection)}
}
func (r *PledgeRepository) Save(ctx context.Context, pledge domain.Pledge) error {
	return saveByID(ctx, r.collection, pledge.PledgeID, pledge)
}
func (r *PledgeRepository) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	return getByID[domain.Pledge](ctx, r.collection, pledgeID)
}
func (r *PledgeRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	items, err := listDocuments[domain.Pledge](ctx, r.collection, bson.M{"shopId": shopID})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
func (r *PledgeRepository) ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
	findOptions := options.Find()
	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}
	cursor, err := r.collection.Find(ctx, bson.M{"chainAnchorStatus": status}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []domain.Pledge
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		var item domain.Pledge
		if err := decodeDocument(doc, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, cursor.Err()
}
