package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	auditsvc "vngrocery/internal/service/audit"
)

type eventLogAdapter struct {
	list func(ctx context.Context, input auditsvc.ListInput) ([]domain.EventLog, error)
}

func (e eventLogAdapter) List(ctx context.Context, input auditsvc.ListInput) ([]domain.EventLog, error) {
	return e.list(ctx, input)
}

func TestEventLogList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) ([]domain.EventLog, error) {
			if input.ResourceType != "shop" || input.ResourceID != "shop-1" || input.ActorUserID != "user-1" {
				t.Fatalf("unexpected list input: %+v", input)
			}
			return []domain.EventLog{{
				EventID:         "event-1",
				ActorUserID:     "user-1",
				ResourceType:    "shop",
				ResourceID:      "shop-1",
				ResourceVersion: 3,
				Action:          "shop.updated",
				Status:          "updated",
				Sequence:        3,
				PreviousEventID: "event-0",
				PayloadJSON:     `{"after":{"name":"Green Shop"}}`,
				PublicKey:       "pub-key",
				KeyAlgorithm:    "Ed25519",
				Signature:       "sig",
				ContentSHA256:   "hash",
				CreatedAt:       now,
			}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop&resourceId=shop-1&actorUserId=user-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload) != 1 || payload[0]["eventId"] != "event-1" {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
}
