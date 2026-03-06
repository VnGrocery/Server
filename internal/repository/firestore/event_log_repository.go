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
