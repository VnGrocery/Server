package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	shopsvc "vngrocery/internal/service/shop"
)

type ShopService interface {
	Create(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error)
	Update(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error)
	Moderate(ctx context.Context, input shopsvc.ModerateInput) (domain.Shop, error)
	GetByID(ctx context.Context, shopID string) (shopsvc.ShopView, error)
	List(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error)
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

	c.JSON(http.StatusCreated, toShopResponse(shopsvc.ShopView{Shop: shop}))
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

	c.JSON(http.StatusOK, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) Moderate(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ModerateShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Moderate(c.Request.Context(), shopsvc.ModerateInput{
		ShopID:          c.Param("shopId"),
		ModeratorUserID: principal.UserID,
		Status:          request.Status,
		ModerationNote:  request.ModerationNote,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) List(c *gin.Context) {
	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.shops.List(c.Request.Context(), shopsvc.ListInput{
		Page:               page,
		PageSize:           pageSize,
		Query:              c.Query("q"),
		Status:             c.Query("status"),
		OwnerUserID:        c.Query("ownerUserId"),
		IncludeAllStatuses: false,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ShopResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toShopResponse(item))
	}
	c.JSON(http.StatusOK, dto.ShopListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
		HasNext:  result.HasNext,
	})
}

func (h *ShopHandler) AdminList(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.shops.List(c.Request.Context(), shopsvc.ListInput{
		Page:               page,
		PageSize:           pageSize,
		Query:              c.Query("q"),
		Status:             c.Query("status"),
		OwnerUserID:        c.Query("ownerUserId"),
		ActorUserID:        principal.UserID,
		IncludeAllStatuses: true,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ShopResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toShopResponse(item))
	}
	c.JSON(http.StatusOK, dto.ShopListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
		HasNext:  result.HasNext,
	})
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
	case errors.Is(err, shopsvc.ErrForbidden), errors.Is(err, shopsvc.ErrAdminRequired):
		status = http.StatusForbidden
	case errors.Is(err, shopsvc.ErrNotFound):
		status = http.StatusNotFound
	}

	c.JSON(status, gin.H{"error": err.Error()})
}

func toShopResponse(view shopsvc.ShopView) dto.ShopResponse {
	shop := view.Shop
	return dto.ShopResponse{
		ShopID:            shop.ShopID,
		OwnerUserID:       shop.OwnerUserID,
		Name:              shop.Name,
		Description:       shop.Description,
		Address:           shop.Address,
		Latitude:          shop.Latitude,
		Longitude:         shop.Longitude,
		Status:            shop.Status,
		ModeratedByUserID: shop.ModeratedByUserID,
		ModerationNote:    shop.ModerationNote,
		ModeratedAt:       shop.ModeratedAt,
		TrustSummary: dto.ShopTrustSummaryResponse{
			HasPledges:         view.TrustSummary.HasPledges,
			PledgeCount:        view.TrustSummary.PledgeCount,
			LatestPledgeID:     view.TrustSummary.LatestPledgeID,
			LatestPledgeStatus: view.TrustSummary.LatestPledgeStatus,
			LatestScore:        view.TrustSummary.LatestScore,
			LatestCategory:     view.TrustSummary.LatestCategory,
			LatestConfidence:   view.TrustSummary.LatestConfidence,
			LastCommittedAt:    view.TrustSummary.LastCommittedAt,
		},
		CreatedAt: shop.CreatedAt,
		UpdatedAt: shop.UpdatedAt,
	}
}

func parsePagination(pageValue, pageSizeValue string) (int, int, error) {
	page := 1
	pageSize := 20
	var err error
	if pageValue != "" {
		page, err = strconv.Atoi(pageValue)
		if err != nil || page < 1 {
			return 0, 0, errors.New("page must be a positive integer")
		}
	}
	if pageSizeValue != "" {
		pageSize, err = strconv.Atoi(pageSizeValue)
		if err != nil || pageSize < 1 {
			return 0, 0, errors.New("pageSize must be a positive integer")
		}
	}
	return page, pageSize, nil
}
