package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	shopsvc "vngrocery/internal/service/shop"
)

type shopServiceStub struct {
	create     func(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error)
	update     func(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error)
	getByID    func(ctx context.Context, shopID string) (domain.Shop, error)
	listActive func(ctx context.Context) ([]domain.Shop, error)
}

func (s shopServiceStub) Create(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error) {
	return s.create(ctx, input)
}

func (s shopServiceStub) Update(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error) {
	return s.update(ctx, input)
}

func (s shopServiceStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return s.getByID(ctx, shopID)
}

func (s shopServiceStub) ListActive(ctx context.Context) ([]domain.Shop, error) {
	return s.listActive(ctx)
}

func TestCreateShop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	handler := NewShopHandler(shopServiceStub{
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
		update: func(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		listActive: func(ctx context.Context) ([]domain.Shop, error) {
			return nil, errors.New("not implemented")
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

func TestUpdateShopRejectsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceStub{
		create: func(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		update: func(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error) {
			return domain.Shop{}, shopsvc.ErrForbidden
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		listActive: func(ctx context.Context) ([]domain.Shop, error) {
			return nil, errors.New("not implemented")
		},
	})

	router := gin.New()
	router.PUT("/v1/shops/:shopId", func(c *gin.Context) {
		c.Set("auth.principal", authservice.Principal{UserID: "user-2"})
		handler.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/v1/shops/shop-1", bytes.NewBufferString(`{"name":"Green Shop","description":"Fresh daily","address":"123 Main St","latitude":10.7,"longitude":106.6}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestListShops(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewShopHandler(shopServiceStub{
		create: func(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		update: func(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not implemented")
		},
		listActive: func(ctx context.Context) ([]domain.Shop, error) {
			return []domain.Shop{{ShopID: "shop-1", Name: "Green Shop", Address: "123 Main St", Latitude: 10.7, Longitude: 106.6, Status: shopsvc.ShopStatusActive}}, nil
		},
	})

	router := gin.New()
	router.GET("/v1/shops", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/shops", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
