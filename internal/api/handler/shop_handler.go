package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	shopsvc "vngrocery/internal/service/shop"
)

type ShopService interface {
	Create(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error)
	Update(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error)
	GetByID(ctx context.Context, shopID string) (domain.Shop, error)
	ListActive(ctx context.Context) ([]domain.Shop, error)
}

type ShopHandler struct {
	shops ShopService
}

func NewShopHandler(shops ShopService) *ShopHandler {
	return &ShopHandler{shops: shops}
}

func (h *ShopHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Create(c.Request.Context(), shopsvc.CreateInput{
		OwnerUserID: principal.UserID,
		Name:        request.Name,
		Description: request.Description,
		Address:     request.Address,
		Latitude:    request.Latitude,
		Longitude:   request.Longitude,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toShopResponse(shop))
}

func (h *ShopHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Update(c.Request.Context(), shopsvc.UpdateInput{
		ShopID:      c.Param("shopId"),
		OwnerUserID: principal.UserID,
		Name:        request.Name,
		Description: request.Description,
		Address:     request.Address,
		Latitude:    request.Latitude,
		Longitude:   request.Longitude,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shop))
}

func (h *ShopHandler) List(c *gin.Context) {
	shops, err := h.shops.ListActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]dto.ShopResponse, 0, len(shops))
	for _, shop := range shops {
		response = append(response, toShopResponse(shop))
	}
	c.JSON(http.StatusOK, response)
}

func (h *ShopHandler) GetByID(c *gin.Context) {
	shop, err := h.shops.GetByID(c.Request.Context(), c.Param("shopId"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shop))
}

func (h *ShopHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, shopsvc.ErrInvalidShop):
		status = http.StatusBadRequest
	case errors.Is(err, shopsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, shopsvc.ErrNotFound):
		status = http.StatusNotFound
	}

	c.JSON(status, gin.H{"error": err.Error()})
}

func toShopResponse(shop domain.Shop) dto.ShopResponse {
	return dto.ShopResponse{
		ShopID:      shop.ShopID,
		OwnerUserID: shop.OwnerUserID,
		Name:        shop.Name,
		Description: shop.Description,
		Address:     shop.Address,
		Latitude:    shop.Latitude,
		Longitude:   shop.Longitude,
		Status:      shop.Status,
		CreatedAt:   shop.CreatedAt,
		UpdatedAt:   shop.UpdatedAt,
	}
}
