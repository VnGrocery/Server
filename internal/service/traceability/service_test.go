package traceability

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	batchsvc "vngrocery/internal/service/batch"
	productsvc "vngrocery/internal/service/product"
	shopsvc "vngrocery/internal/service/shop"
)

func TestCreateTraceEvent(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	var saved domain.TraceEvent
	service := newTraceService(traceEventRepoStub{
		save: func(ctx context.Context, event domain.TraceEvent) error {
			saved = event
			return nil
		},
	})
	service.now = func() time.Time { return now }

	event, err := service.CreateTraceEvent(context.Background(), CreateTraceEventInput{
		ShopID:      "shop-1",
		ProductID:   "product-1",
		BatchID:     "batch-1",
		ActorUserID: "seller-1",
		Type:        "origin",
		Title:       "Trang trại A",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if event.EventID == "" || saved.EventID != event.EventID || event.Status != StatusActive {
		t.Fatalf("unexpected event: %#v saved=%#v", event, saved)
	}
	if !event.OccurredAt.Equal(now) {
		t.Fatalf("expected default occurredAt, got %v", event.OccurredAt)
	}
}

func TestCreateTraceEventRejectsNonOwner(t *testing.T) {
	service := newTraceService(traceEventRepoStub{
		save: func(ctx context.Context, event domain.TraceEvent) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.CreateTraceEvent(context.Background(), CreateTraceEventInput{
		ShopID:      "shop-1",
		ProductID:   "product-1",
		BatchID:     "batch-1",
		ActorUserID: "seller-2",
		Type:        "origin",
		Title:       "Trang trại A",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestListTraceEventsFiltersActivePublicEvents(t *testing.T) {
	service := newTraceService(traceEventRepoStub{
		list: func(ctx context.Context, filter repository.TraceEventListFilter) ([]domain.TraceEvent, error) {
			if filter.ShopID != "shop-1" || filter.ProductID != "product-1" || filter.BatchID != "batch-1" || filter.Status != StatusActive {
				t.Fatalf("unexpected filter: %#v", filter)
			}
			return []domain.TraceEvent{{EventID: "event-1", Status: StatusActive}}, nil
		},
	})

	events, err := service.ListTraceEvents(context.Background(), ListTraceEventsInput{
		ShopID:    "shop-1",
		ProductID: "product-1",
		BatchID:   "batch-1",
		Public:    true,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(events) != 1 || events[0].EventID != "event-1" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func newTraceService(events traceEventRepoStub) *Service {
	return NewService(events, batchRepoStub{}, productRepoStub{}, shopRepoStub{})
}

type traceEventRepoStub struct {
	save func(ctx context.Context, event domain.TraceEvent) error
	list func(ctx context.Context, filter repository.TraceEventListFilter) ([]domain.TraceEvent, error)
}

func (s traceEventRepoStub) Save(ctx context.Context, event domain.TraceEvent) error {
	if s.save != nil {
		return s.save(ctx, event)
	}
	return nil
}

func (s traceEventRepoStub) GetByID(ctx context.Context, eventID string) (domain.TraceEvent, error) {
	return domain.TraceEvent{}, nil
}

func (s traceEventRepoStub) List(ctx context.Context, filter repository.TraceEventListFilter) ([]domain.TraceEvent, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, nil
}

type batchRepoStub struct{}

func (batchRepoStub) Save(ctx context.Context, batch domain.ProductBatch) error { return nil }
func (batchRepoStub) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-1", OwnerUserID: "seller-1", Status: batchsvc.StatusActive}, nil
}
func (batchRepoStub) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	return nil, nil
}

type productRepoStub struct{}

func (productRepoStub) Save(ctx context.Context, product domain.Product) error { return nil }
func (productRepoStub) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-1", Status: productsvc.ProductStatusActive}, nil
}
func (productRepoStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	return nil, nil
}

type shopRepoStub struct{}

func (shopRepoStub) Save(ctx context.Context, shop domain.Shop) error { return nil }
func (shopRepoStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
}
func (shopRepoStub) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	return nil, nil
}
