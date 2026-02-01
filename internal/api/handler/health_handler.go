package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, dto.HealthResponse{
		Status: "ok",
	})
}
