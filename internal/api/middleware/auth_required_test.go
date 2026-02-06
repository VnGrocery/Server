package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authservice "vngrocery/internal/service/auth"
)

type stubVerifier struct {
	verify func(ctx context.Context, token string) (authservice.Principal, error)
}

func (s stubVerifier) Verify(ctx context.Context, token string) (authservice.Principal, error) {
	return s.verify(ctx, token)
}

func TestAuthRequiredRejectsMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := NewAuthRequired(stubVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			t.Fatal("verify should not be called when authorization header is missing")
			return authservice.Principal{}, nil
		},
	})

	router := gin.New()
	router.GET("/protected", middleware.Handle(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthRequiredRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := NewAuthRequired(stubVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	})

	router := gin.New()
	router.GET("/protected", middleware.Handle(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthRequiredStoresPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expected := authservice.Principal{
		UserID: "user-1",
		Email:  "seller@example.com",
	}

	middleware := NewAuthRequired(stubVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			if token != "good-token" {
				return authservice.Principal{}, errors.New("unexpected token value")
			}
			return expected, nil
		},
	})

	router := gin.New()
	router.GET("/protected", middleware.Handle(), func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			t.Fatal("principal should exist in request context")
		}

		c.JSON(http.StatusOK, principal)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var actual authservice.Principal
	if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if actual != expected {
		t.Fatalf("expected %+v, got %+v", expected, actual)
	}
}
