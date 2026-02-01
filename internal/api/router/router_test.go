package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	authservice "vngrocery/internal/service/auth"
)

type testVerifier struct {
	verify func(ctx context.Context, token string) (authservice.Principal, error)
}

func (t testVerifier) Verify(ctx context.Context, token string) (authservice.Principal, error) {
	return t.verify(ctx, token)
}

func TestRouterHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, nil
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterMeUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterMeAuthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{UserID: "user-123", Email: "u@example.com"}, nil
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
