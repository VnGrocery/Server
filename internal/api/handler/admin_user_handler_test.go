package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	useradminsvc "vngrocery/internal/service/useradmin"
)

type adminUserServiceAdapter struct {
	list         func(ctx context.Context, input useradminsvc.ListInput) ([]domain.User, error)
	updateRole   func(ctx context.Context, input useradminsvc.UpdateRoleInput) (domain.User, error)
	updateStatus func(ctx context.Context, input useradminsvc.UpdateStatusInput) (domain.User, error)
}

func (a adminUserServiceAdapter) List(ctx context.Context, input useradminsvc.ListInput) ([]domain.User, error) {
	return a.list(ctx, input)
}

func (a adminUserServiceAdapter) UpdateRole(ctx context.Context, input useradminsvc.UpdateRoleInput) (domain.User, error) {
	return a.updateRole(ctx, input)
}

func (a adminUserServiceAdapter) UpdateStatus(ctx context.Context, input useradminsvc.UpdateStatusInput) (domain.User, error) {
	return a.updateStatus(ctx, input)
}

func TestUpdateUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAdminUserHandler(adminUserServiceAdapter{
		updateRole: func(ctx context.Context, input useradminsvc.UpdateRoleInput) (domain.User, error) {
			if input.ActorUserID != "admin-1" || input.TargetUserID != "user-1" || input.ExpectedVersion != 2 || input.Role != "admin" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return domain.User{UserID: input.TargetUserID, Role: input.Role, Status: "active", Version: 3}, nil
		},
	})

	router := gin.New()
	router.PATCH("/v1/admin/users/:userId/role", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "admin-1"})
		handler.UpdateRole(c)
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/user-1/role", bytes.NewBufferString(`{"expectedVersion":2,"role":"admin"}`))
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
	if payload["role"] != "admin" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestListAdminUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAdminUserHandler(adminUserServiceAdapter{
		list: func(ctx context.Context, input useradminsvc.ListInput) ([]domain.User, error) {
			if input.ActorUserID != "admin-1" || input.Status != "active" || input.Role != "user" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return []domain.User{{UserID: "user-1", Role: "user", Status: "active", Version: 1}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/admin/users", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "admin-1"})
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?status=active&role=user", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUpdateUserStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAdminUserHandler(adminUserServiceAdapter{
		updateStatus: func(ctx context.Context, input useradminsvc.UpdateStatusInput) (domain.User, error) {
			if input.ActorUserID != "admin-1" || input.TargetUserID != "user-1" || input.ExpectedVersion != 2 || input.Status != "suspended" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return domain.User{UserID: input.TargetUserID, Role: "user", Status: input.Status, Version: 3}, nil
		},
	})

	router := gin.New()
	router.PATCH("/v1/admin/users/:userId/status", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "admin-1"})
		handler.UpdateStatus(c)
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/user-1/status", bytes.NewBufferString(`{"expectedVersion":2,"status":"suspended"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
