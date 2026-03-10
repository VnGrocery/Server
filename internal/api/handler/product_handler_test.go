package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	productsvc "vngrocery/internal/service/product"
)

type productServiceAdapter struct {
	create  func(ctx context.Context, input productsvc.CreateInput) (domain.Product, error)
	update  func(ctx context.Context, input productsvc.UpdateInput) (domain.Product, error)
	delete  func(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error)
	getByID func(ctx context.Context, shopID, productID string) (domain.Product, error)
	list    func(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error)
}

func (s productServiceAdapter) Create(ctx context.Context, input productsvc.CreateInput) (domain.Product, error) {
	return s.create(ctx, input)
}

func (s productServiceAdapter) Update(ctx context.Context, input productsvc.UpdateInput) (domain.Product, error) {
	return s.update(ctx, input)
}

func (s productServiceAdapter) Delete(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error) {
	return s.delete(ctx, input)
}

func (s productServiceAdapter) GetByID(ctx context.Context, shopID, productID string) (domain.Product, error) {
	return s.getByID(ctx, shopID, productID)
}

func (s productServiceAdapter) List(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error) {
	return s.list(ctx, input)
}

func TestCreateProduct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	handler := NewProductHandler(productServiceAdapter{
		create: func(ctx context.Context, input productsvc.CreateInput) (domain.Product, error) {
			if input.ShopID != "shop-1" || input.OwnerUserID != "user-1" {
				t.Fatalf("unexpected create input: %+v", input)
			}
			return domain.Product{
				ProductID:   "product-1",
				ShopID:      input.ShopID,
				OwnerUserID: input.OwnerUserID,
				Name:        input.Name,
				Description: input.Description,
				Price:       input.Price,
				Currency:    input.Currency,
				Status:      productsvc.ProductStatusActive,
				Version:     1,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops/:shopId/products", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/products", bytes.NewBufferString(`{"name":"Apple","description":"Fresh","price":10000,"currency":"VND"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestDeleteProductRequiresExpectedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductHandler(productServiceAdapter{
		delete: func(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error) {
			t.Fatal("delete should not be called")
			return domain.Product{}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/shops/:shopId/products/:productId", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/shops/shop-1/products/product-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListProducts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductHandler(productServiceAdapter{
		list: func(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error) {
			if input.ShopID != "shop-1" {
				t.Fatalf("unexpected list input: %+v", input)
			}
			return []domain.Product{{
				ProductID:   "product-1",
				ShopID:      input.ShopID,
				OwnerUserID: "user-1",
				Name:        "Apple",
				Price:       10000,
				Currency:    "VND",
				Status:      productsvc.ProductStatusActive,
				Version:     1,
			}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/products", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
