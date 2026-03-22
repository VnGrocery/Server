package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	shopsvc "vngrocery/internal/service/shop"
)

func TestCreateShop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	handler := NewShopHandler(shopServiceAdapter{
		create: func(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error) {
			if input.OwnerUserID != "user-1" {
				t.Fatalf("unexpected ownerUserID: %s", input.OwnerUserID)
			}
			return domain.Shop{
				ShopID:      "shop-1",
				OwnerUserID: input.OwnerUserID,
				Name:        input.Name,
				Description: input.Description,
				Address:     input.Address,
				Latitude:    input.Latitude,
				Longitude:   input.Longitude,
				Status:      shopsvc.ShopStatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops", bytes.NewBufferString(`{"name":"Green Shop","description":"Fresh daily","address":"123 Main St","latitude":10.7,"longitude":106.6}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestListShopsWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		list: func(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error) {
			if input.Page != 2 || input.PageSize != 1 || input.Query != "green" {
				t.Fatalf("unexpected list input: %+v", input)
			}
			return shopsvc.ListResult{
				Items: []shopsvc.ShopView{{
					Shop: domain.Shop{
						ShopID:    "shop-1",
						Name:      "Green Shop",
						Address:   "123 Main St",
						Latitude:  10.7,
						Longitude: 106.6,
						Status:    shopsvc.ShopStatusActive,
					},
					TrustSummary: shopsvc.TrustSummary{
						HasPledges:  true,
						PledgeCount: 1,
					},
				}},
				Page:     2,
				PageSize: 1,
				Total:    3,
				HasNext:  true,
			}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops?page=2&pageSize=1&q=green", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["page"] != float64(2) {
		t.Fatalf("unexpected page: %v", payload["page"])
	}
}

func TestModerateShopRejectsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		moderate: func(ctx context.Context, input shopsvc.ModerateInput) (domain.Shop, error) {
			return domain.Shop{}, shopsvc.ErrAdminRequired
		},
	})

	router := gin.New()
	router.PATCH("/v1/admin/shops/:shopId/moderation", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-2"})
		handler.Moderate(c)
	})

	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/shops/shop-1/moderation", bytes.NewBufferString(`{"status":"suspended","moderationNote":"fraud review"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestDeleteShop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		deleteFn: func(ctx context.Context, input shopsvc.DeleteInput) (domain.Shop, error) {
			if input.ShopID != "shop-1" || input.OwnerUserID != "user-1" || input.ExpectedVersion != 3 {
				t.Fatalf("unexpected delete input: %+v", input)
			}
			return domain.Shop{ShopID: input.ShopID, OwnerUserID: input.OwnerUserID, Status: shopsvc.ShopStatusDeleted}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/shops/:shopId", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/shops/shop-1?expectedVersion=3", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestDeleteShopRequiresExpectedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		deleteFn: func(ctx context.Context, input shopsvc.DeleteInput) (domain.Shop, error) {
			t.Fatal("delete should not be called")
			return domain.Shop{}, nil
		},
	})

	router := gin.New()
	router.DELETE("/v1/shops/:shopId", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/shops/shop-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateReview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)
	handler := NewShopHandler(shopServiceAdapter{
		review: func(ctx context.Context, input shopsvc.ReviewInput) (domain.ShopReview, error) {
			if input.ShopID != "shop-1" || input.ReviewerUserID != "user-1" || input.Rating != 5 {
				t.Fatalf("unexpected review input: %+v", input)
			}
			return domain.ShopReview{
				ReviewID:       "review-1",
				ShopID:         input.ShopID,
				ReviewerUserID: input.ReviewerUserID,
				Rating:         input.Rating,
				Comment:        input.Comment,
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/shops/:shopId/reviews", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-1"})
		handler.CreateReview(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/reviews", bytes.NewBufferString(`{"rating":5,"comment":"Great shop"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestListReviews(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		listReviews: func(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
			return []domain.ShopReview{{ReviewID: "review-1", ShopID: shopID, ReviewerUserID: "user-1", Rating: 4, Comment: "Good"}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/reviews", handler.ListReviews)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/reviews", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListPledgesFiltersForBuyerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceAdapter{
		listPledges: func(ctx context.Context, input shopsvc.PledgeHistoryInput) ([]domain.Pledge, error) {
			if input.ShopID != "shop-1" || input.ProductID != "product-1" || input.Category != "fresh_produce" {
				t.Fatalf("unexpected pledge history input: %#v", input)
			}
			return []domain.Pledge{{PledgeID: "pledge-1", ShopID: input.ShopID, ProductID: input.ProductID, Category: input.Category, Score: 8.5}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops/:shopId/pledges", handler.ListPledges)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/pledges?productId=product-1&category=fresh_produce", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload dto.PledgeHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].PledgeID != "pledge-1" {
		t.Fatalf("unexpected pledge response: %#v", payload)
	}
}

type shopServiceAdapter struct {
	create       func(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error)
	update       func(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error)
	deleteFn     func(ctx context.Context, input shopsvc.DeleteInput) (domain.Shop, error)
	getByID      func(ctx context.Context, shopID string) (shopsvc.ShopView, error)
	list         func(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error)
	listPledges  func(ctx context.Context, input shopsvc.PledgeHistoryInput) ([]domain.Pledge, error)
	moderate     func(ctx context.Context, input shopsvc.ModerateInput) (domain.Shop, error)
	review       func(ctx context.Context, input shopsvc.ReviewInput) (domain.ShopReview, error)
	deleteReview func(ctx context.Context, input shopsvc.DeleteReviewInput) (domain.ShopReview, error)
	listReviews  func(ctx context.Context, shopID string) ([]domain.ShopReview, error)
}

func (s shopServiceAdapter) Create(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error) {
	return s.create(ctx, input)
}
func (s shopServiceAdapter) Update(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error) {
	if s.update == nil {
		return domain.Shop{}, errors.New("not implemented")
	}
	return s.update(ctx, input)
}
func (s shopServiceAdapter) Delete(ctx context.Context, input shopsvc.DeleteInput) (domain.Shop, error) {
	if s.deleteFn == nil {
		return domain.Shop{}, errors.New("not implemented")
	}
	return s.deleteFn(ctx, input)
}
func (s shopServiceAdapter) Moderate(ctx context.Context, input shopsvc.ModerateInput) (domain.Shop, error) {
	return s.moderate(ctx, input)
}
func (s shopServiceAdapter) GetByID(ctx context.Context, shopID string) (shopsvc.ShopView, error) {
	if s.getByID == nil {
		return shopsvc.ShopView{}, errors.New("not implemented")
	}
	return s.getByID(ctx, shopID)
}
func (s shopServiceAdapter) List(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error) {
	return s.list(ctx, input)
}
func (s shopServiceAdapter) ListPledges(ctx context.Context, input shopsvc.PledgeHistoryInput) ([]domain.Pledge, error) {
	if s.listPledges == nil {
		return nil, errors.New("not implemented")
	}
	return s.listPledges(ctx, input)
}
func (s shopServiceAdapter) Review(ctx context.Context, input shopsvc.ReviewInput) (domain.ShopReview, error) {
	return s.review(ctx, input)
}
func (s shopServiceAdapter) DeleteReview(ctx context.Context, input shopsvc.DeleteReviewInput) (domain.ShopReview, error) {
	if s.deleteReview == nil {
		return domain.ShopReview{}, errors.New("not implemented")
	}
	return s.deleteReview(ctx, input)
}
func (s shopServiceAdapter) ListReviews(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	return s.listReviews(ctx, shopID)
}
