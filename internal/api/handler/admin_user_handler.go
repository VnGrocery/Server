package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	useradminsvc "vngrocery/internal/service/useradmin"
)

type AdminUserService interface {
	List(ctx context.Context, input useradminsvc.ListInput) ([]domain.User, error)
	UpdateRole(ctx context.Context, input useradminsvc.UpdateRoleInput) (domain.User, error)
	UpdateStatus(ctx context.Context, input useradminsvc.UpdateStatusInput) (domain.User, error)
	RotateAccountKey(ctx context.Context, input useradminsvc.AccountKeyInput) (useradminsvc.AccountKeyResult, error)
	RecoverAccountKey(ctx context.Context, input useradminsvc.AccountKeyInput) (useradminsvc.AccountKeyResult, error)
	BackfillAccountKey(ctx context.Context, input useradminsvc.AccountKeyInput) (useradminsvc.AccountKeyResult, error)
}

type AdminUserHandler struct {
	users AdminUserService
}

func NewAdminUserHandler(users AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{users: users}
}

func (h *AdminUserHandler) List(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	users, err := h.users.List(c.Request.Context(), useradminsvc.ListInput{
		ActorUserID: principal.UserID,
		Status:      c.Query("status"),
		Role:        c.Query("role"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.AdminUserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, toAdminUserResponse(user))
	}
	c.JSON(http.StatusOK, dto.AdminUserListResponse{Items: items})
}

func (h *AdminUserHandler) UpdateRole(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	user, err := h.users.UpdateRole(c.Request.Context(), useradminsvc.UpdateRoleInput{
		ActorUserID:     principal.UserID,
		TargetUserID:    c.Param("userId"),
		ExpectedVersion: request.ExpectedVersion,
		Role:            request.Role,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAdminUserResponse(user))
}

func (h *AdminUserHandler) UpdateStatus(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	user, err := h.users.UpdateStatus(c.Request.Context(), useradminsvc.UpdateStatusInput{
		ActorUserID:     principal.UserID,
		TargetUserID:    c.Param("userId"),
		ExpectedVersion: request.ExpectedVersion,
		Status:          request.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAdminUserResponse(user))
}

func (h *AdminUserHandler) RotateAccountKey(c *gin.Context) {
	h.accountKeyAction(c, h.users.RotateAccountKey, "rotate")
}

func (h *AdminUserHandler) RecoverAccountKey(c *gin.Context) {
	h.accountKeyAction(c, h.users.RecoverAccountKey, "recover")
}

func (h *AdminUserHandler) BackfillAccountKey(c *gin.Context) {
	h.accountKeyAction(c, h.users.BackfillAccountKey, "backfill")
}

func (h *AdminUserHandler) accountKeyAction(c *gin.Context, action func(context.Context, useradminsvc.AccountKeyInput) (useradminsvc.AccountKeyResult, error), mode string) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.AccountKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := action(c.Request.Context(), useradminsvc.AccountKeyInput{
		ActorUserID:     principal.UserID,
		TargetUserID:    c.Param("userId"),
		ExpectedVersion: request.ExpectedVersion,
		Mode:            mode,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AccountKeyResponse{
		UserID:       result.UserID,
		PublicKey:    result.PublicKey,
		KeyAlgorithm: result.KeyAlgorithm,
		VaultKeyPath: result.VaultKeyPath,
		Version:      result.Version,
	})
}

func (h *AdminUserHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, useradminsvc.ErrInvalidUser):
		status = http.StatusBadRequest
	case errors.Is(err, useradminsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, useradminsvc.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, useradminsvc.ErrVersionConflict):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toAdminUserResponse(user domain.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		UserID:      user.UserID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		Version:     user.Version,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
