package firestore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type TraceEventRepository struct {
	client *gofirestore.Client
}

func NewTraceEventRepository(client *gofirestore.Client) *TraceEventRepository {
	return &TraceEventRepository{client: client}
}

func (r *TraceEventRepository) Save(ctx context.Context, event domain.TraceEvent) error {
	_, err := r.client.Collection(TraceEventsCollection).Doc(event.EventID).Set(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to save trace event: %w", err)
	}
	return nil
}

func (r *TraceEventRepository) GetByID(ctx context.Context, eventID string) (domain.TraceEvent, error) {
	doc, err := r.client.Collection(TraceEventsCollection).Doc(eventID).Get(ctx)
	if err != nil {
		return domain.TraceEvent{}, fmt.Errorf("failed to get trace event: %w", err)
	}
	var event domain.TraceEvent
	if err := doc.DataTo(&event); err != nil {
		return domain.TraceEvent{}, fmt.Errorf("failed to decode trace event document: %w", err)
	}
	return event, nil
}

func (r *TraceEventRepository) List(ctx context.Context, filter repository.TraceEventListFilter) ([]domain.TraceEvent, error) {
	docs, err := r.client.Collection(TraceEventsCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list trace events: %w", err)
	}
	events := make([]domain.TraceEvent, 0, len(docs))
	for _, doc := range docs {
		var event domain.TraceEvent
		if err := doc.DataTo(&event); err != nil {
			return nil, fmt.Errorf("failed to decode trace event document: %w", err)
		}
		if strings.TrimSpace(filter.ShopID) != "" && event.ShopID != strings.TrimSpace(filter.ShopID) {
			continue
		}
		if strings.TrimSpace(filter.ProductID) != "" && event.ProductID != strings.TrimSpace(filter.ProductID) {
			continue
		}
		if strings.TrimSpace(filter.BatchID) != "" && event.BatchID != strings.TrimSpace(filter.BatchID) {
			continue
		}
		if strings.TrimSpace(filter.Type) != "" && event.Type != strings.TrimSpace(filter.Type) {
			continue
		}
		if strings.TrimSpace(filter.Status) != "" && event.Status != strings.TrimSpace(filter.Status) {
			continue
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].OccurredAt.Before(events[j].OccurredAt) })
	return events, nil
}
