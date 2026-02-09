package shop

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
)

type shopRepositoryStub struct {
	save       func(ctx context.Context, shop domain.Shop) error
	getByID    func(ctx context.Context, shopID string) (domain.Shop, error)
	listActive func(ctx context.Context) ([]domain.Shop, error)
}

func (s shopRepositoryStub) Save(ctx context.Context, shop domain.Shop) error {
	return s.save(ctx, shop)
}

func (s shopRepositoryStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return s.getByID(ctx, shopID)
}

func (s shopRepositoryStub) ListActive(ctx context.Context) ([]domain.Shop, error) {
	if s.listActive == nil {
		return nil, errors.New("not implemented")
	}
	return s.listActive(ctx)
}

func TestCreateShop(t *testing.T) {
	fixedTime := time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			if shop.ShopID == "" {
				t.Fatal("expected generated shop id")
			}
			if shop.OwnerUserID != "user-1" {
				t.Fatalf("unexpected owner: %s", shop.OwnerUserID)
			}
			if shop.Status != ShopStatusActive {
				t.Fatalf("unexpected status: %s", shop.Status)
			}
			if !shop.CreatedAt.Equal(fixedTime) || !shop.UpdatedAt.Equal(fixedTime) {
				t.Fatal("expected fixed timestamps")
			}
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	})
	service.now = func() time.Time { return fixedTime }

	shop, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: "user-1",
		Name:        "Green Shop",
		Description: "Fresh daily",
		Address:     "123 Main St",
		Latitude:    10.762622,
		Longitude:   106.660172,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shop.ShopID == "" {
		t.Fatal("expected shop id")
	}
}

func TestUpdateRejectsNonOwner(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "owner-1"}, nil
		},
	})

	_, err := service.Update(context.Background(), UpdateInput{
		ShopID:      "shop-1",
		OwnerUserID: "user-2",
		Name:        "Name",
		Address:     "Address",
		Latitude:    10,
		Longitude:   106,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateRejectsInvalidCoordinates(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	})

	_, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: "user-1",
		Name:        "Shop",
		Address:     "Address",
		Latitude:    200,
		Longitude:   106,
	})
	if !errors.Is(err, ErrInvalidShop) {
		t.Fatalf("expected ErrInvalidShop, got %v", err)
	}
}
