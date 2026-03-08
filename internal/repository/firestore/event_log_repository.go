package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type EventLogRepository struct {
	client *gofirestore.Client
}

func NewEventLogRepository(client *gofirestore.Client) *EventLogRepository {
	return &EventLogRepository{client: client}
}

func (r *EventLogRepository) Save(ctx context.Context, event domain.EventLog) error {
	_, err := r.client.Collection(EventLogsCollection).Doc(event.EventID).Set(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to save event log: %w", err)
	}
	return nil
}

func (r *EventLogRepository) GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
	docs, err := r.client.Collection(EventLogsCollection).
		Where("resourceType", "==", resourceType).
		Where("resourceId", "==", resourceID).
		Documents(ctx).
		GetAll()
	if err != nil {
		return domain.EventLog{}, fmt.Errorf("failed to query event logs: %w", err)
	}
	if len(docs) == 0 {
		return domain.EventLog{}, nil
	}

	var latest domain.EventLog
	for _, doc := range docs {
		var event domain.EventLog
		if err := doc.DataTo(&event); err != nil {
			return domain.EventLog{}, fmt.Errorf("failed to decode event log: %w", err)
		}
		if event.Sequence > latest.Sequence {
			latest = event
		}
	}
	return latest, nil
}
