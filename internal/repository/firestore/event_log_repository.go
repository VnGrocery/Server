package firestore

import (
	"context"
	"fmt"
	"sort"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
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

func (r *EventLogRepository) GetByID(ctx context.Context, eventID string) (domain.EventLog, error) {
	doc, err := r.client.Collection(EventLogsCollection).Doc(eventID).Get(ctx)
	if err != nil {
		return domain.EventLog{}, fmt.Errorf("failed to get event log: %w", err)
	}

	var event domain.EventLog
	if err := doc.DataTo(&event); err != nil {
		return domain.EventLog{}, fmt.Errorf("failed to decode event log: %w", err)
	}
	return event, nil
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

func (r *EventLogRepository) List(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
	query := r.client.Collection(EventLogsCollection).Query
	if filter.ResourceType != "" {
		query = query.Where("resourceType", "==", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query = query.Where("resourceId", "==", filter.ResourceID)
	}
	if filter.ActorUserID != "" {
		query = query.Where("actorUserId", "==", filter.ActorUserID)
	}
	if filter.Action != "" {
		query = query.Where("action", "==", filter.Action)
	}
	if filter.Status != "" {
		query = query.Where("status", "==", filter.Status)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list event logs: %w", err)
	}

	events := make([]domain.EventLog, 0, len(docs))
	for _, doc := range docs {
		var event domain.EventLog
		if err := doc.DataTo(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event log: %w", err)
		}
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
		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].Sequence > events[j].Sequence
		}
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})

	return events, nil
}
