package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authservice "vngrocery/internal/service/auth"
)

type authAccountAdapter struct {
	refresh              func(ctx context.Context, refreshToken string) (authservice.AuthResult, error)
	logout               func(ctx context.Context, refreshToken string) error
	changePassword       func(ctx context.Context, userID, currentPassword, newPassword string) error
	requestPasswordReset func(ctx context.Context, email string) (authservice.PasswordResetResult, error)
	resetPassword        func(ctx context.Context, resetToken, newPassword string) error
	delete               func(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error)
}

func (a authAccountAdapter) Register(ctx context.Context, email, password, displayName string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, nil
}

func (a authAccountAdapter) Login(ctx context.Context, email, password string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, nil
}

func (a authAccountAdapter) GoogleLogin(ctx context.Context, googleIDToken string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, nil
}

func (a authAccountAdapter) Refresh(ctx context.Context, refreshToken string) (authservice.AuthResult, error) {
	return a.refresh(ctx, refreshToken)
}

func (a authAccountAdapter) Logout(ctx context.Context, refreshToken string) error {
	return a.logout(ctx, refreshToken)
}

func (a authAccountAdapter) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return a.changePassword(ctx, userID, currentPassword, newPassword)
}

func (a authAccountAdapter) RequestPasswordReset(ctx context.Context, email string) (authservice.PasswordResetResult, error) {
	return a.requestPasswordReset(ctx, email)
}

func (a authAccountAdapter) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	return a.resetPassword(ctx, resetToken, newPassword)
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

func TestRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(authAccountAdapter{
		refresh: func(ctx context.Context, refreshToken string) (authservice.AuthResult, error) {
			if refreshToken != "refresh-token" {
				t.Fatalf("unexpected refresh token: %s", refreshToken)
			}
			return authservice.AuthResult{
				AccessToken:  "access-token",
				RefreshToken: "next-refresh-token",
				Principal:    authservice.Principal{UserID: "user-1", Email: "u@example.com"},
				PublicKey:    "pub-key",
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/auth/refresh", handler.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewBufferString(`{"refreshToken":"refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["accessToken"] != "access-token" || payload["refreshToken"] != "next-refresh-token" {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(authAccountAdapter{
		logout: func(ctx context.Context, refreshToken string) error {
			if refreshToken != "refresh-token" {
				t.Fatalf("unexpected refresh token: %s", refreshToken)
			}
			return nil
		},
	})

	router := gin.New()
	router.POST("/v1/auth/logout", handler.Logout)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", bytes.NewBufferString(`{"refreshToken":"refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
