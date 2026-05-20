package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	traceabilitysvc "vngrocery/internal/service/traceability"
)

type traceEventServiceAdapter struct {
	create func(ctx context.Context, input traceabilitysvc.CreateTraceEventInput) (domain.TraceEvent, error)
	list   func(ctx context.Context, input traceabilitysvc.ListTraceEventsInput) ([]domain.TraceEvent, error)
}

func (s traceEventServiceAdapter) CreateTraceEvent(ctx context.Context, input traceabilitysvc.CreateTraceEventInput) (domain.TraceEvent, error) {
	return s.create(ctx, input)
}

func (s traceEventServiceAdapter) ListTraceEvents(ctx context.Context, input traceabilitysvc.ListTraceEventsInput) ([]domain.TraceEvent, error) {
	return s.list(ctx, input)
}

func TestCreateTraceEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	handler := NewTraceEventHandler(traceEventServiceAdapter{
		create: func(ctx context.Context, input traceabilitysvc.CreateTraceEventInput) (domain.TraceEvent, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" || input.BatchID != "batch-1" || input.ActorUserID != "seller-1" || input.Type != "origin" {
				t.Fatalf("unexpected input: %#v", input)
			}
			return domain.TraceEvent{
				EventID:     "event-1",
				ShopID:      input.ShopID,
				ProductID:   input.ProductID,
				BatchID:     input.BatchID,
				ActorUserID: input.ActorUserID,
				Type:        input.Type,
				Title:       input.Title,
				Status:      traceabilitysvc.StatusActive,
				OccurredAt:  now,
				CreatedAt:   now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops/:shopId/products/:productId/batches/:batchId/trace-events", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "seller-1"})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/products/product-1/batches/batch-1/trace-events", bytes.NewBufferString(`{"type":"origin","title":"Trang trại A"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListTraceEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTraceEventHandler(traceEventServiceAdapter{
		list: func(ctx context.Context, input traceabilitysvc.ListTraceEventsInput) ([]domain.TraceEvent, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" || input.BatchID != "batch-1" || !input.Public {
				t.Fatalf("unexpected input: %#v", input)
			}
			return []domain.TraceEvent{{EventID: "event-1", Status: traceabilitysvc.StatusActive}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/products/:productId/batches/:batchId/trace-events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/products/product-1/batches/batch-1/trace-events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
