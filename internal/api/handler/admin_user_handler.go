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
	UpdateRole(ctx context.Context, input useradminsvc.UpdateRoleInput) (domain.User, error)
}

type AdminUserHandler struct {
	users AdminUserService
}

func NewAdminUserHandler(users AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{users: users}
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
