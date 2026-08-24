package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	vouchersvc "vngrocery/internal/service/voucher"
)

type VoucherHandler struct{ service *vouchersvc.Service }

func NewVoucherHandler(service *vouchersvc.Service) *VoucherHandler {
	return &VoucherHandler{service: service}
}

func (h *VoucherHandler) Get(c *gin.Context) {
	voucher, err := h.service.Get(c.Request.Context(), c.Param("voucherId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toVoucherResponse(voucher))
}

func (h *VoucherHandler) ListShop(c *gin.Context) {
	items, err := h.service.ListShop(c.Request.Context(), c.Param("shopId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	responses := make([]dto.VoucherResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toVoucherResponse(item))
	}
	c.JSON(http.StatusOK, dto.VoucherListResponse{Items: responses})
}

// ListFeatured serves the home screen's offer slot.
func (h *VoucherHandler) ListFeatured(c *gin.Context) {
	limit := 5
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}
	items, err := h.service.ListFeatured(c.Request.Context(), limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	responses := make([]dto.FeaturedVoucherResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dto.FeaturedVoucherResponse{
			VoucherResponse: toVoucherResponse(item.Voucher),
			ShopName:        item.ShopName,
		})
	}
	c.JSON(http.StatusOK, dto.FeaturedVoucherListResponse{Items: responses})
}

func (h *VoucherHandler) Check(c *gin.Context) {
	var request dto.CheckVoucherRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	result, err := h.service.Check(c.Request.Context(), vouchersvc.CheckInput{Code: request.Code, ShopID: request.ShopID, OrderValue: request.OrderValue})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response := dto.VoucherCheckResponse{Valid: result.Valid, Message: result.Message, DiscountAmount: result.DiscountAmount, FinalPrice: result.FinalPrice}
	if result.Voucher != nil {
		value := toVoucherResponse(*result.Voucher)
		response.Voucher = &value
	}
	c.JSON(http.StatusOK, response)
}

func (h *VoucherHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found"})
		return
	}
	var request dto.CreateVoucherRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	voucher, err := h.service.Create(c.Request.Context(), vouchersvc.CreateInput{ShopID: c.Param("shopId"), OwnerUserID: principal.UserID, Code: request.Code, Title: request.Title, DiscountValue: request.DiscountValue, IsPercent: request.IsPercent, MinSpend: request.MinSpend, ExpiresAt: request.ExpiresAt, Note: request.Note, CodeFormat: request.CodeFormat, TotalQuantity: request.TotalQuantity})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toVoucherResponse(voucher))
}

func (h *VoucherHandler) Wallet(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found"})
		return
	}
	items, err := h.service.Wallet(c.Request.Context(), principal.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	responses := make([]dto.UserVoucherResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toUserVoucherResponse(item))
	}
	c.JSON(http.StatusOK, dto.UserVoucherListResponse{Items: responses})
}

func (h *VoucherHandler) Save(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found"})
		return
	}
	var request dto.SaveVoucherRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	item, err := h.service.SaveToWallet(c.Request.Context(), principal.UserID, request.VoucherID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserVoucherResponse(item))
}

func (h *VoucherHandler) AddManual(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found"})
		return
	}
	var request dto.ManualVoucherRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	item, err := h.service.AddManual(c.Request.Context(), principal.UserID, vouchersvc.CreateInput{ShopID: request.ShopID, Code: request.Code, Title: request.Title, Note: request.Note, CodeFormat: request.CodeFormat, ExpiresAt: request.ExpiresAt})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserVoucherResponse(item))
}

func (h *VoucherHandler) Use(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found"})
		return
	}
	item, err := h.service.Use(c.Request.Context(), principal.UserID, c.Param("userVoucherId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserVoucherResponse(item))
}

func (h *VoucherHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, vouchersvc.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, vouchersvc.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, vouchersvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, vouchersvc.ErrSoldOut),
		errors.Is(err, vouchersvc.ErrExpired),
		errors.Is(err, vouchersvc.ErrUsed):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toVoucherResponse(v domain.Voucher) dto.VoucherResponse {
	return dto.VoucherResponse{
		VoucherID: v.VoucherID, ShopID: v.ShopID, Code: v.Code, Title: v.Title,
		DiscountValue: v.DiscountValue, IsPercent: v.IsPercent, MinSpend: v.MinSpend,
		ExpiresAt: v.ExpiresAt, Active: v.Active, Manual: v.Manual, Note: v.Note,
		CodeFormat:    v.CodeFormat,
		TotalQuantity: v.TotalQuantity,
		ClaimedCount:  v.ClaimedCount,
	}
}

func toUserVoucherResponse(item vouchersvc.WalletItem) dto.UserVoucherResponse {
	return dto.UserVoucherResponse{UserVoucherID: item.UserVoucher.UserVoucherID, VoucherID: item.UserVoucher.VoucherID, Used: item.UserVoucher.Used, UsedAt: item.UserVoucher.UsedAt, Voucher: toVoucherResponse(item.Voucher)}
}
