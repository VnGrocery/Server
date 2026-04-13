package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
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

type stubUserRepo struct {
	getByID func(ctx context.Context, userID string) (domain.User, error)
}

func (s stubUserRepo) Save(ctx context.Context, user domain.User) error { return nil }
func (s stubUserRepo) GetByID(ctx context.Context, userID string) (domain.User, error) {
	return s.getByID(ctx, userID)
}
func (s stubUserRepo) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	return nil, nil
}

func TestAdminRequiredRejectsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := NewAdminRequired(stubUserRepo{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "seller"}, nil
		},
	})

	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		c.Set(principalContextKey, authservice.Principal{UserID: "user-1"})
		middleware.Handle()(c)
	}, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAdminRequiredAllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := NewAdminRequired(stubUserRepo{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "admin"}, nil
		},
	})

	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		c.Set(principalContextKey, authservice.Principal{UserID: "user-1"})
		middleware.Handle()(c)
	}, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
