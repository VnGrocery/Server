package router

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	authservice "vngrocery/internal/service/auth"
	visionservice "vngrocery/internal/service/vision"
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
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}),
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
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}),
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
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}),
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

func TestRouterSellerScoreProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

type sellerScorerStub struct{}

func (sellerScorerStub) Score(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
	return visionservice.ScoreResult{
		Score:      8.3,
		Category:   "fresh_produce",
		Confidence: 0.91,
	}, nil
}
