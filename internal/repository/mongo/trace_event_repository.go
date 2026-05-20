package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type TraceEventRepository struct{ collection *mongo.Collection }

func NewTraceEventRepository(db *mongo.Database) *TraceEventRepository {
	return &TraceEventRepository{collection: db.Collection(traceEventsCollection)}
}

func (r *TraceEventRepository) Save(ctx context.Context, event domain.TraceEvent) error {
	return saveByID(ctx, r.collection, event.EventID, event)
}

func (r *TraceEventRepository) GetByID(ctx context.Context, eventID string) (domain.TraceEvent, error) {
	return getByID[domain.TraceEvent](ctx, r.collection, eventID)
}

func (r *TraceEventRepository) List(ctx context.Context, filter repository.TraceEventListFilter) ([]domain.TraceEvent, error) {
	query := bson.M{}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.ProductID != "" {
		query["productId"] = filter.ProductID
	}
	if filter.BatchID != "" {
		query["batchId"] = filter.BatchID
	}
	if filter.Type != "" {
		query["type"] = filter.Type
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	items, err := listDocuments[domain.TraceEvent](ctx, r.collection, query)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.Before(items[j].OccurredAt) })
	return items, nil
}
