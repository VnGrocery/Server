package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	batchsvc "vngrocery/internal/service/batch"
)

type productBatchServiceAdapter struct {
	create  func(ctx context.Context, input batchsvc.CreateInput) (domain.ProductBatch, error)
	update  func(ctx context.Context, input batchsvc.UpdateInput) (domain.ProductBatch, error)
	delete  func(ctx context.Context, input batchsvc.DeleteInput) (domain.ProductBatch, error)
	getByID func(ctx context.Context, shopID, productID, batchID string) (domain.ProductBatch, error)
	list    func(ctx context.Context, input batchsvc.ListInput) ([]domain.ProductBatch, error)
}

func (s productBatchServiceAdapter) Create(ctx context.Context, input batchsvc.CreateInput) (domain.ProductBatch, error) {
	return s.create(ctx, input)
}

func (s productBatchServiceAdapter) Update(ctx context.Context, input batchsvc.UpdateInput) (domain.ProductBatch, error) {
	return s.update(ctx, input)
}

func (s productBatchServiceAdapter) Delete(ctx context.Context, input batchsvc.DeleteInput) (domain.ProductBatch, error) {
	return s.delete(ctx, input)
}

func (s productBatchServiceAdapter) GetByID(ctx context.Context, shopID, productID, batchID string) (domain.ProductBatch, error) {
	return s.getByID(ctx, shopID, productID, batchID)
}

func (s productBatchServiceAdapter) List(ctx context.Context, input batchsvc.ListInput) ([]domain.ProductBatch, error) {
	return s.list(ctx, input)
}

func TestCreateProductBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	handler := NewProductBatchHandler(productBatchServiceAdapter{
		create: func(ctx context.Context, input batchsvc.CreateInput) (domain.ProductBatch, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" || input.OwnerUserID != "seller-1" {
				t.Fatalf("unexpected create input: %+v", input)
			}
			return domain.ProductBatch{
				BatchID:          "batch-1",
				ShopID:           input.ShopID,
				ProductID:        input.ProductID,
				OwnerUserID:      input.OwnerUserID,
				BatchCode:        input.BatchCode,
				CurrentFreshness: input.CurrentFreshness,
				Status:           batchsvc.StatusActive,
				Version:          1,
				CreatedAt:        now,
				UpdatedAt:        now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops/:shopId/products/:productId/batches", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "seller-1"})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/products/product-1/batches", bytes.NewBufferString(`{"batchCode":"BATCH-001","currentFreshness":91}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProductBatchRequiresExpectedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductBatchHandler(productBatchServiceAdapter{
		delete: func(ctx context.Context, input batchsvc.DeleteInput) (domain.ProductBatch, error) {
			t.Fatal("delete should not be called")
			return domain.ProductBatch{}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/shops/:shopId/products/:productId/batches/:batchId", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "seller-1"})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/shops/shop-1/products/product-1/batches/batch-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListProductBatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductBatchHandler(productBatchServiceAdapter{
		list: func(ctx context.Context, input batchsvc.ListInput) ([]domain.ProductBatch, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" {
				t.Fatalf("unexpected list input: %+v", input)
			}
			return []domain.ProductBatch{{BatchID: "batch-1", ShopID: input.ShopID, ProductID: input.ProductID, Status: batchsvc.StatusActive}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/products/:productId/batches", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/products/product-1/batches", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
