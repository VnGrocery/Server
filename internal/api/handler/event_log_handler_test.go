package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vngrocery/internal/domain"
	auditsvc "vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
)

type eventLogAdapter struct {
	list           func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error)
	verifyEvent    func(ctx context.Context, input auditsvc.VerifyEventInput) (auditsvc.EventVerificationResult, error)
	verifyResource func(ctx context.Context, input auditsvc.VerifyResourceInput) (auditsvc.VerifyResourceResult, error)
}

func (e eventLogAdapter) List(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
	return e.list(ctx, input)
}

func (e eventLogAdapter) VerifyEvent(ctx context.Context, input auditsvc.VerifyEventInput) (auditsvc.EventVerificationResult, error) {
	if e.verifyEvent == nil {
		return auditsvc.EventVerificationResult{}, nil
	}
	return e.verifyEvent(ctx, input)
}

func (e eventLogAdapter) VerifyResource(ctx context.Context, input auditsvc.VerifyResourceInput) (auditsvc.VerifyResourceResult, error) {
	if e.verifyResource == nil {
		return auditsvc.VerifyResourceResult{}, nil
	}
	return e.verifyResource(ctx, input)
}

func TestEventLogList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 7, 2, 0, 0, 0, time.UTC)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			if input.ResourceType != "shop" || input.ResourceID != "shop-1" || input.ActorUserID != "user-1" {
				t.Fatalf("unexpected list input: %+v", input)
			}
			if input.Action != "shop.updated" || input.Status != "updated" || input.MinSequence != 2 || input.MaxSequence != 4 {
				t.Fatalf("unexpected extended filters: %+v", input)
			}
			if !input.CreatedAfter.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) ||
				!input.CreatedBefore.Equal(time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)) ||
				input.Page != 2 || input.PageSize != 1 {
				t.Fatalf("unexpected pagination/time filters: %+v", input)
			}
			return auditsvc.ListResult{
				Items: []domain.EventLog{{
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
				}},
				Total:    3,
				Page:     2,
				PageSize: 1,
			}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "admin-1", Role: "admin"})
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop&resourceId=shop-1&actorUserId=user-1&action=shop.updated&status=updated&minSequence=2&maxSequence=4&createdAfter=2026-04-01T00:00:00Z&createdBefore=2026-04-08T00:00:00Z&page=2&pageSize=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected response items: %#v", payload)
	}
	first, ok := items[0].(map[string]any)
	if !ok || first["eventId"] != "event-1" {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	pagination, ok := payload["pagination"].(map[string]any)
	if !ok || pagination["page"] != float64(2) || pagination["pageSize"] != float64(1) || pagination["totalItems"] != float64(3) || pagination["totalPages"] != float64(3) {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
}

func TestEventLogListRejectsInvalidTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			t.Fatalf("list should not be called on invalid query")
			return auditsvc.ListResult{}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?createdAfter=bad-time", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEventLogListReturnsTooManyRequestsOnQuotaExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			return auditsvc.ListResult{}, fmt.Errorf("failed to list event logs: %w", status.Error(codes.ResourceExhausted, "Quota exceeded"))
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestEventLogListReturnsForbiddenOnPermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			return auditsvc.ListResult{}, fmt.Errorf("failed to list event logs: %w", status.Error(codes.PermissionDenied, "forbidden"))
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestEventLogListReturnsGatewayTimeoutOnDeadlineExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			return auditsvc.ListResult{}, fmt.Errorf("failed to list event logs: %w", context.DeadlineExceeded)
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", rec.Code)
	}
}

func TestEventLogVerifyEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		verifyEvent: func(ctx context.Context, input auditsvc.VerifyEventInput) (auditsvc.EventVerificationResult, error) {
			if input.EventID != "event-1" {
				t.Fatalf("unexpected verify input: %+v", input)
			}
			return auditsvc.EventVerificationResult{
				EventID:              "event-1",
				ResourceType:         "shop",
				ResourceID:           "shop-1",
				Sequence:             2,
				PreviousEventID:      "event-0",
				ContentHashValid:     true,
				SignatureValid:       true,
				ChainLinkValid:       true,
				PreviousEventPresent: true,
				Verified:             true,
			}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events/:eventId/verify", handler.VerifyEvent)

	req := httptest.NewRequest(http.MethodGet, "/v1/events/event-1/verify", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestEventLogVerifyResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		verifyResource: func(ctx context.Context, input auditsvc.VerifyResourceInput) (auditsvc.VerifyResourceResult, error) {
			if input.ResourceType != "shop" || input.ResourceID != "shop-1" {
				t.Fatalf("unexpected verify input: %+v", input)
			}
			return auditsvc.VerifyResourceResult{
				ResourceType: "shop",
				ResourceID:   "shop-1",
				EventCount:   2,
				Verified:     true,
				Events: []auditsvc.EventVerificationResult{{
					EventID:          "event-1",
					ResourceType:     "shop",
					ResourceID:       "shop-1",
					Sequence:         1,
					ContentHashValid: true,
					SignatureValid:   true,
					ChainLinkValid:   true,
					Verified:         true,
				}},
			}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events/verify", handler.VerifyResource)

	req := httptest.NewRequest(http.MethodGet, "/v1/events/verify?resourceType=shop&resourceId=shop-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMapEventLogListErrorFallbackBadRequest(t *testing.T) {
	statusCode, message := mapEventLogListError(fmt.Errorf("invalid filter"))
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", statusCode)
	}
	if message != "invalid filter" {
		t.Fatalf("unexpected message: %s", message)
	}
}

func TestMapEventLogListErrorTimeoutUnwrap(t *testing.T) {
	statusCode, message := mapEventLogListError(fmt.Errorf("wrapped: %w", context.DeadlineExceeded))
	if statusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", statusCode)
	}
	if message == "" {
		t.Fatalf("expected timeout message")
	}
}

func TestListResponseBodyIsJson(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			return auditsvc.ListResult{}, fmt.Errorf("failed to list event logs: %w", status.Error(codes.ResourceExhausted, "Quota exceeded"))
		},
	})

	router := gin.New()
	router.GET("/v1/events", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?resourceType=shop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	if len(body) == 0 || body[0] != '{' {
		t.Fatalf("expected json body, got %q", string(body))
	}
}

func TestEventLogListClampsActorToCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var asked string
	handler := NewEventLogHandler(eventLogAdapter{
		list: func(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error) {
			asked = input.ActorUserID
			return auditsvc.ListResult{Page: 1, PageSize: 20}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/events", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1", Role: "user"})
		handler.List(c)
	})

	// Asking for somebody else's trail returns the caller's own, not theirs.
	req := httptest.NewRequest(http.MethodGet, "/v1/events?actorUserId=user-2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if asked != "user-1" {
		t.Fatalf("expected the caller's own id, got %q", asked)
	}
}
