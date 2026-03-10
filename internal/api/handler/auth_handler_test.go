package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authservice "vngrocery/internal/service/auth"
)

type authAccountAdapter struct {
	delete func(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error)
}

func (a authAccountAdapter) Register(ctx context.Context, email, password, displayName string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", nil
}

func (a authAccountAdapter) Login(ctx context.Context, email, password string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", nil
}

func (a authAccountAdapter) GoogleLogin(ctx context.Context, googleIDToken string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", nil
}

func (a authAccountAdapter) Delete(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error) {
	return a.delete(ctx, userID, expectedVersion)
}

func TestDeleteMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(authAccountAdapter{
		delete: func(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error) {
			if userID != "user-1" || expectedVersion != 2 {
				t.Fatalf("unexpected delete input userID=%s expectedVersion=%d", userID, expectedVersion)
			}
			return authservice.DeleteResult{UserID: userID, Status: authservice.AccountStatusDeleted}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/me", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.DeleteMe(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/me?expectedVersion=2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["status"] != authservice.AccountStatusDeleted {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
}

func TestDeleteMeRequiresExpectedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(authAccountAdapter{
		delete: func(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error) {
			t.Fatal("delete should not be called")
			return authservice.DeleteResult{}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/me", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.DeleteMe(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
