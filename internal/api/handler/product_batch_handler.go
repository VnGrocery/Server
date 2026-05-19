package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	batchsvc "vngrocery/internal/service/batch"
)

type ProductBatchService interface {
	Create(ctx context.Context, input batchsvc.CreateInput) (domain.ProductBatch, error)
	Update(ctx context.Context, input batchsvc.UpdateInput) (domain.ProductBatch, error)
	Delete(ctx context.Context, input batchsvc.DeleteInput) (domain.ProductBatch, error)
	GetByID(ctx context.Context, shopID, productID, batchID string) (domain.ProductBatch, error)
	List(ctx context.Context, input batchsvc.ListInput) ([]domain.ProductBatch, error)
}

type ProductBatchHandler struct {
	batches ProductBatchService
}

func NewProductBatchHandler(batches ProductBatchService) *ProductBatchHandler {
	return &ProductBatchHandler{batches: batches}
}

func (h *ProductBatchHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.CreateProductBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	batch, err := h.batches.Create(c.Request.Context(), batchsvc.CreateInput{
		ShopID:           c.Param("shopId"),
		ProductID:        c.Param("productId"),
		OwnerUserID:      principal.UserID,
		BatchCode:        request.BatchCode,
		OriginName:       request.OriginName,
		OriginAddress:    request.OriginAddress,
		SupplierName:     request.SupplierName,
		SlaughteredAt:    request.SlaughteredAt,
		PackedAt:         request.PackedAt,
		ReceivedAt:       request.ReceivedAt,
		ExpiresAt:        request.ExpiresAt,
		Quantity:         request.Quantity,
		QuantityUnit:     request.QuantityUnit,
		StorageTempMin:   request.StorageTempMin,
		StorageTempMax:   request.StorageTempMax,
		CurrentFreshness: request.CurrentFreshness,
		CurrentCategory:  request.CurrentCategory,
		Status:           request.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProductBatchResponse(batch))
}

func (h *ProductBatchHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.UpdateProductBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	batch, err := h.batches.Update(c.Request.Context(), batchsvc.UpdateInput{
		BatchID:          c.Param("batchId"),
		ShopID:           c.Param("shopId"),
		ProductID:        c.Param("productId"),
		OwnerUserID:      principal.UserID,
		ExpectedVersion:  request.ExpectedVersion,
		BatchCode:        request.BatchCode,
		OriginName:       request.OriginName,
		OriginAddress:    request.OriginAddress,
		SupplierName:     request.SupplierName,
		SlaughteredAt:    request.SlaughteredAt,
		PackedAt:         request.PackedAt,
		ReceivedAt:       request.ReceivedAt,
		ExpiresAt:        request.ExpiresAt,
		Quantity:         request.Quantity,
		QuantityUnit:     request.QuantityUnit,
		StorageTempMin:   request.StorageTempMin,
		StorageTempMax:   request.StorageTempMax,
		CurrentFreshness: request.CurrentFreshness,
		CurrentCategory:  request.CurrentCategory,
		Status:           request.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductBatchResponse(batch))
}

func (h *ProductBatchHandler) Delete(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	expectedVersion, parseErr := parsePositiveIntQuery(c.Query("expectedVersion"), "expectedVersion")
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}
	batch, err := h.batches.Delete(c.Request.Context(), batchsvc.DeleteInput{
		BatchID:         c.Param("batchId"),
		ShopID:          c.Param("shopId"),
		ProductID:       c.Param("productId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductBatchResponse(batch))
}

func (h *ProductBatchHandler) GetByID(c *gin.Context) {
	batch, err := h.batches.GetByID(c.Request.Context(), c.Param("shopId"), c.Param("productId"), c.Param("batchId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductBatchResponse(batch))
}

func (h *ProductBatchHandler) List(c *gin.Context) {
	batches, err := h.batches.List(c.Request.Context(), batchsvc.ListInput{
		ShopID:    c.Param("shopId"),
		ProductID: c.Param("productId"),
		Status:    c.Query("status"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]dto.ProductBatchResponse, 0, len(batches))
	for _, batch := range batches {
		items = append(items, toProductBatchResponse(batch))
	}
	c.JSON(http.StatusOK, dto.ProductBatchListResponse{Items: items})
}

func (h *ProductBatchHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, batchsvc.ErrInvalidBatch):
		status = http.StatusBadRequest
	case errors.Is(err, batchsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, batchsvc.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, batchsvc.ErrVersionConflict):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toProductBatchResponse(batch domain.ProductBatch) dto.ProductBatchResponse {
	return dto.ProductBatchResponse{
		BatchID:          batch.BatchID,
		ProductID:        batch.ProductID,
		ShopID:           batch.ShopID,
		OwnerUserID:      batch.OwnerUserID,
		BatchCode:        batch.BatchCode,
		OriginName:       batch.OriginName,
		OriginAddress:    batch.OriginAddress,
		SupplierName:     batch.SupplierName,
		SlaughteredAt:    batch.SlaughteredAt,
		PackedAt:         batch.PackedAt,
		ReceivedAt:       batch.ReceivedAt,
		ExpiresAt:        batch.ExpiresAt,
		Quantity:         batch.Quantity,
		QuantityUnit:     batch.QuantityUnit,
		StorageTempMin:   batch.StorageTempMin,
		StorageTempMax:   batch.StorageTempMax,
		CurrentFreshness: batch.CurrentFreshness,
		CurrentCategory:  batch.CurrentCategory,
		Status:           batch.Status,
		Version:          batch.Version,
		CreatedAt:        batch.CreatedAt,
		UpdatedAt:        batch.UpdatedAt,
	}
}
