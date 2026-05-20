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
	create                    func(ctx context.Context, input productsvc.CreateInput) (domain.Product, error)
	update                    func(ctx context.Context, input productsvc.UpdateInput) (domain.Product, error)
	delete                    func(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error)
	moderate                  func(ctx context.Context, input productsvc.ModerateInput) (domain.Product, error)
	bulkUpsert                func(ctx context.Context, input productsvc.BulkUpsertInput) ([]domain.Product, error)
	getByID                   func(ctx context.Context, shopID, productID string) (domain.Product, error)
	list                      func(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error)
	createFreshnessReport     func(ctx context.Context, input productsvc.FreshnessReportInput) (domain.ProductFreshnessReport, error)
	moderateFreshnessReport   func(ctx context.Context, input productsvc.ModerateFreshnessReportInput) (domain.ProductFreshnessReport, error)
	listFreshnessReports      func(ctx context.Context, shopID, productID, batchID string) ([]domain.ProductFreshnessReport, error)
	listFreshnessReportsAdmin func(ctx context.Context, input productsvc.ListFreshnessReportAdminInput) (productsvc.ListFreshnessReportAdminResult, error)
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

func (s productServiceAdapter) Moderate(ctx context.Context, input productsvc.ModerateInput) (domain.Product, error) {
	return s.moderate(ctx, input)
}

func (s productServiceAdapter) BulkUpsert(ctx context.Context, input productsvc.BulkUpsertInput) ([]domain.Product, error) {
	return s.bulkUpsert(ctx, input)
}

func (s productServiceAdapter) GetByID(ctx context.Context, shopID, productID string) (domain.Product, error) {
	return s.getByID(ctx, shopID, productID)
}

func (s productServiceAdapter) List(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error) {
	return s.list(ctx, input)
}

func (s productServiceAdapter) CreateFreshnessReport(ctx context.Context, input productsvc.FreshnessReportInput) (domain.ProductFreshnessReport, error) {
	return s.createFreshnessReport(ctx, input)
}

func (s productServiceAdapter) ModerateFreshnessReport(ctx context.Context, input productsvc.ModerateFreshnessReportInput) (domain.ProductFreshnessReport, error) {
	return s.moderateFreshnessReport(ctx, input)
}

func (s productServiceAdapter) ListFreshnessReports(ctx context.Context, shopID, productID, batchID string) ([]domain.ProductFreshnessReport, error) {
	return s.listFreshnessReports(ctx, shopID, productID, batchID)
}

func (s productServiceAdapter) ListFreshnessReportsAdmin(ctx context.Context, input productsvc.ListFreshnessReportAdminInput) (productsvc.ListFreshnessReportAdminResult, error) {
	if s.listFreshnessReportsAdmin == nil {
		return productsvc.ListFreshnessReportAdminResult{}, nil
	}
	return s.listFreshnessReportsAdmin(ctx, input)
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

func TestCreateProductFreshnessReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 7, 11, 0, 0, 0, time.UTC)
	handler := NewProductHandler(productServiceAdapter{
		createFreshnessReport: func(ctx context.Context, input productsvc.FreshnessReportInput) (domain.ProductFreshnessReport, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" || input.ReporterUserID != "buyer-1" {
				t.Fatalf("unexpected report input: %+v", input)
			}
			return domain.ProductFreshnessReport{
				ReportID:       "report-1",
				ShopID:         input.ShopID,
				ProductID:      input.ProductID,
				BatchID:        input.BatchID,
				ReporterUserID: input.ReporterUserID,
				Status:         productsvc.FreshnessReportStatusActive,
				Version:        1,
				Score:          input.Score,
				Category:       input.Category,
				Confidence:     input.Confidence,
				Comment:        input.Comment,
				ImageHash:      input.ImageHash,
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops/:shopId/products/:productId/freshness-reports", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "buyer-1"})
		handler.CreateFreshnessReport(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/products/product-1/freshness-reports", bytes.NewBufferString(`{"batchId":"batch-1","score":6.2,"category":"bruised_fruit","confidence":0.86,"comment":"Bruised","imageHash":"hash-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestListProductFreshnessReports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductHandler(productServiceAdapter{
		listFreshnessReports: func(ctx context.Context, shopID, productID, batchID string) ([]domain.ProductFreshnessReport, error) {
			if shopID != "shop-1" || productID != "product-1" || batchID != "batch-1" {
				t.Fatalf("unexpected list report input: shop=%s product=%s batch=%s", shopID, productID, batchID)
			}
			return []domain.ProductFreshnessReport{{ReportID: "report-1", ShopID: shopID, ProductID: productID, Status: productsvc.FreshnessReportStatusActive}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/products/:productId/freshness-reports", handler.ListFreshnessReports)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/products/product-1/freshness-reports?batchId=batch-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListFreshnessReportsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProductHandler(productServiceAdapter{
		listFreshnessReportsAdmin: func(ctx context.Context, input productsvc.ListFreshnessReportAdminInput) (productsvc.ListFreshnessReportAdminResult, error) {
			if input.ActorUserID != "admin-1" || input.Status != "active" {
				t.Fatalf("unexpected admin list input: %#v", input)
			}
			return productsvc.ListFreshnessReportAdminResult{
				Items: []domain.ProductFreshnessReport{{
					ReportID:       "report-1",
					ShopID:         "shop-1",
					ProductID:      "product-1",
					ReporterUserID: "buyer-1",
					Status:         productsvc.FreshnessReportStatusActive,
					Version:        1,
				}},
				Page:     1,
				PageSize: 20,
				Total:    1,
			}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/admin/product-freshness-reports", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "admin-1"})
		handler.ListFreshnessReportsAdmin(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/product-freshness-reports?status=active&page=1&pageSize=20", nil)
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
