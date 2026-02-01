package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Me(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "khong tim thay thong tin user da xac thuc",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MeResponse{
		UserID:      principal.UserID,
		Email:       principal.Email,
		PhoneNumber: principal.PhoneNumber,
	})
}
