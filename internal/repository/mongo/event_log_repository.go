package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type EventLogRepository struct{ collection *mongo.Collection }

func NewEventLogRepository(db *mongo.Database) *EventLogRepository {
	return &EventLogRepository{collection: db.Collection(eventLogsCollection)}
}
func (r *EventLogRepository) Save(ctx context.Context, event domain.EventLog) error {
	return saveByID(ctx, r.collection, event.EventID, event)
}
func (r *EventLogRepository) GetByID(ctx context.Context, eventID string) (domain.EventLog, error) {
	return getByID[domain.EventLog](ctx, r.collection, eventID)
}
func (r *EventLogRepository) GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
	items, err := listDocuments[domain.EventLog](ctx, r.collection, bson.M{"resourceType": resourceType, "resourceId": resourceID})
	if err != nil {
		return domain.EventLog{}, err
	}
	if len(items) == 0 {
		return domain.EventLog{}, nil
	}
	latest := items[0]
	for _, item := range items[1:] {
		if item.Sequence > latest.Sequence {
			latest = item
		}
	}
	return latest, nil
}
func (r *EventLogRepository) List(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
	query := bson.M{}
	if filter.ResourceType != "" {
		query["resourceType"] = filter.ResourceType
	}
	if filter.ResourceID != "" {
		query["resourceId"] = filter.ResourceID
	}
	if filter.ActorUserID != "" {
		query["actorUserId"] = filter.ActorUserID
	}
	if filter.Action != "" {
		query["action"] = filter.Action
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	items, err := listDocuments[domain.EventLog](ctx, r.collection, query)
	if err != nil {
		return nil, err
	}
	out := make([]domain.EventLog, 0, len(items))
	for _, event := range items {
		if filter.MinSequence > 0 && event.Sequence < filter.MinSequence {
			continue
		}
		if filter.MaxSequence > 0 && event.Sequence > filter.MaxSequence {
			continue
		}
		if !filter.CreatedAfter.IsZero() && event.CreatedAt.Before(filter.CreatedAfter) {
			continue
		}
		if !filter.CreatedBefore.IsZero() && event.CreatedAt.After(filter.CreatedBefore) {
			continue
		}
		out = append(out, event)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].Sequence > out[j].Sequence
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
