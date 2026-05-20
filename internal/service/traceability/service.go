package traceability

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	batchsvc "vngrocery/internal/service/batch"
	shopsvc "vngrocery/internal/service/shop"
)

var (
	ErrInvalidTraceEvent = errors.New("invalid trace event request")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("trace resource not found")
)

const (
	StatusActive = "active"
	StatusHidden = "hidden"
)

const (
	EventTypeOrigin         = "origin"
	EventTypeSlaughter      = "slaughter"
	EventTypePackaging      = "packaging"
	EventTypeShipping       = "shipping"
	EventTypeReceived       = "received"
	EventTypeStorageCheck   = "storage_check"
	EventTypeFreshnessCheck = "freshness_check"
	EventTypeRecall         = "recall"
	EventTypeDisposal       = "disposal"
)

const maxFutureOccurrenceSkew = 5 * time.Minute

type CreateTraceEventInput struct {
	BatchID      string
	ProductID    string
	ShopID       string
	ActorUserID  string
	Type         string
	Title        string
	Description  string
	LocationName string
	Latitude     float64
	Longitude    float64
	Temperature  *float64
	Humidity     *float64
	ImageCID     string
	ImageHash    string
	DataHash     string
	OccurredAt   time.Time
}

type ListTraceEventsInput struct {
	BatchID   string
	ProductID string
	ShopID    string
	Type      string
	Public    bool
}

type Service struct {
	events   repository.TraceEventRepository
	batches  repository.ProductBatchRepository
	products repository.ProductRepository
	shops    repository.ShopRepository
	now      func() time.Time
}

func NewService(events repository.TraceEventRepository, batches repository.ProductBatchRepository, products repository.ProductRepository, shops repository.ShopRepository) *Service {
	return &Service{
		events:   events,
		batches:  batches,
		products: products,
		shops:    shops,
		now:      time.Now,
	}
}

func (s *Service) CreateTraceEvent(ctx context.Context, input CreateTraceEventInput) (domain.TraceEvent, error) {
	if s.events == nil || s.batches == nil || s.products == nil || s.shops == nil {
		return domain.TraceEvent{}, fmt.Errorf("traceability dependencies are not configured")
	}
	if strings.TrimSpace(input.ShopID) == "" || strings.TrimSpace(input.ProductID) == "" || strings.TrimSpace(input.BatchID) == "" || strings.TrimSpace(input.ActorUserID) == "" {
		return domain.TraceEvent{}, fmt.Errorf("%w: shopId, productId, batchId and actorUserId are required", ErrInvalidTraceEvent)
	}
	eventType := strings.TrimSpace(input.Type)
	if err := validateTraceEventType(eventType, true); err != nil {
		return domain.TraceEvent{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.TraceEvent{}, fmt.Errorf("%w: title is required", ErrInvalidTraceEvent)
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = s.now().UTC()
	}
	occurredAt := input.OccurredAt.UTC()
	if occurredAt.After(s.now().UTC().Add(maxFutureOccurrenceSkew)) {
		return domain.TraceEvent{}, fmt.Errorf("%w: occurredAt is too far in the future", ErrInvalidTraceEvent)
	}
	batch, err := s.requireOwnedBatch(ctx, input.ShopID, input.ProductID, input.BatchID, input.ActorUserID)
	if err != nil {
		return domain.TraceEvent{}, err
	}

	now := s.now().UTC()
	event := domain.TraceEvent{
		EventID:      uuid.NewString(),
		BatchID:      strings.TrimSpace(input.BatchID),
		ProductID:    strings.TrimSpace(input.ProductID),
		ShopID:       strings.TrimSpace(input.ShopID),
		ActorUserID:  strings.TrimSpace(input.ActorUserID),
		Type:         eventType,
		Title:        strings.TrimSpace(input.Title),
		Description:  strings.TrimSpace(input.Description),
		LocationName: strings.TrimSpace(input.LocationName),
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		Temperature:  input.Temperature,
		Humidity:     input.Humidity,
		ImageCID:     strings.TrimSpace(input.ImageCID),
		ImageHash:    strings.TrimSpace(input.ImageHash),
		DataHash:     strings.TrimSpace(input.DataHash),
		Status:       StatusActive,
		OccurredAt:   occurredAt,
		CreatedAt:    now,
	}
	if err := s.events.Save(ctx, event); err != nil {
		return domain.TraceEvent{}, err
	}
	if event.Type == EventTypeRecall {
		batch.Status = batchsvc.StatusRecalled
		batch.Version++
		batch.UpdatedAt = now
		if err := s.batches.Save(ctx, batch); err != nil {
			return domain.TraceEvent{}, err
		}
	}
	return event, nil
}

func (s *Service) ListTraceEvents(ctx context.Context, input ListTraceEventsInput) ([]domain.TraceEvent, error) {
	if s.events == nil || s.batches == nil || s.products == nil || s.shops == nil {
		return nil, fmt.Errorf("traceability dependencies are not configured")
	}
	if strings.TrimSpace(input.ShopID) == "" || strings.TrimSpace(input.ProductID) == "" || strings.TrimSpace(input.BatchID) == "" {
		return nil, fmt.Errorf("%w: shopId, productId and batchId are required", ErrInvalidTraceEvent)
	}
	if err := s.requirePublicBatch(ctx, input.ShopID, input.ProductID, input.BatchID); err != nil {
		return nil, err
	}
	eventType := strings.TrimSpace(input.Type)
	if err := validateTraceEventType(eventType, false); err != nil {
		return nil, err
	}
	status := ""
	if input.Public {
		status = StatusActive
	}
	events, err := s.events.List(ctx, repository.TraceEventListFilter{
		ShopID:    strings.TrimSpace(input.ShopID),
		ProductID: strings.TrimSpace(input.ProductID),
		BatchID:   strings.TrimSpace(input.BatchID),
		Type:      eventType,
		Status:    status,
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].OccurredAt.Before(events[j].OccurredAt)
	})
	return events, nil
}

func validateTraceEventType(eventType string, required bool) error {
	if strings.TrimSpace(eventType) == "" {
		if required {
			return fmt.Errorf("%w: type is required", ErrInvalidTraceEvent)
		}
		return nil
	}
	switch strings.TrimSpace(eventType) {
	case EventTypeOrigin, EventTypeSlaughter, EventTypePackaging, EventTypeShipping, EventTypeReceived, EventTypeStorageCheck, EventTypeFreshnessCheck, EventTypeRecall, EventTypeDisposal:
		return nil
	default:
		return fmt.Errorf("%w: unsupported trace event type", ErrInvalidTraceEvent)
	}
}

func (s *Service) requireOwnedBatch(ctx context.Context, shopID, productID, batchID, actorUserID string) (domain.ProductBatch, error) {
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.ProductBatch{}, ErrNotFound
	}
	if shop.OwnerUserID != strings.TrimSpace(actorUserID) {
		return domain.ProductBatch{}, ErrForbidden
	}
	product, err := s.products.GetByID(ctx, strings.TrimSpace(productID))
	if err != nil || product.ProductID == "" || product.ShopID != strings.TrimSpace(shopID) {
		return domain.ProductBatch{}, ErrNotFound
	}
	batch, err := s.batches.GetByID(ctx, strings.TrimSpace(batchID))
	if err != nil || batch.BatchID == "" || batch.Status == batchsvc.StatusDeleted || batch.ShopID != strings.TrimSpace(shopID) || batch.ProductID != strings.TrimSpace(productID) {
		return domain.ProductBatch{}, ErrNotFound
	}
	if batch.OwnerUserID != "" && batch.OwnerUserID != strings.TrimSpace(actorUserID) {
		return domain.ProductBatch{}, ErrForbidden
	}
	return batch, nil
}

func (s *Service) requirePublicBatch(ctx context.Context, shopID, productID, batchID string) error {
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return ErrNotFound
	}
	product, err := s.products.GetByID(ctx, strings.TrimSpace(productID))
	if err != nil || product.ProductID == "" || product.ShopID != strings.TrimSpace(shopID) {
		return ErrNotFound
	}
	batch, err := s.batches.GetByID(ctx, strings.TrimSpace(batchID))
	if err != nil || batch.BatchID == "" || batch.Status == batchsvc.StatusDeleted || batch.ShopID != strings.TrimSpace(shopID) || batch.ProductID != strings.TrimSpace(productID) {
		return ErrNotFound
	}
	return nil
}
