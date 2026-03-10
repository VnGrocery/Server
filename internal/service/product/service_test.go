package product

import (
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

type productRepositoryStub struct {
	save    func(ctx context.Context, product domain.Product) error
	getByID func(ctx context.Context, productID string) (domain.Product, error)
	list    func(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error)
}

func (s productRepositoryStub) Save(ctx context.Context, product domain.Product) error {
	if s.save != nil {
		return s.save(ctx, product)
	}
	return nil
}

func (s productRepositoryStub) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	if s.getByID != nil {
		return s.getByID(ctx, productID)
	}
	return domain.Product{}, nil
}

func (s productRepositoryStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, nil
}

type shopRepositoryStub struct {
	getByID func(ctx context.Context, shopID string) (domain.Shop, error)
}

func (s shopRepositoryStub) Save(ctx context.Context, shop domain.Shop) error { return nil }
func (s shopRepositoryStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	if s.getByID != nil {
		return s.getByID(ctx, shopID)
	}
	return domain.Shop{}, nil
}
func (s shopRepositoryStub) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	return nil, nil
}

type auditLoggerStub struct {
	logHits int
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	return nil
}

func TestCreateProduct(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	service := NewService(productRepositoryStub{
		save: func(ctx context.Context, product domain.Product) error {
			if product.Status != ProductStatusActive || product.Version != 1 {
				t.Fatalf("unexpected saved product: %#v", product)
			}
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1", Status: "active"}, nil
		},
	}, auditLogger)

	product, err := service.Create(context.Background(), CreateInput{
		ShopID:      "shop-1",
		OwnerUserID: "user-1",
		Name:        "Apple",
		Price:       10000,
		Currency:    "vnd",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if product.Currency != "VND" {
		t.Fatalf("unexpected currency: %s", product.Currency)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestUpdateProductRejectsVersionConflict(t *testing.T) {
	service := NewService(productRepositoryStub{
		save: func(ctx context.Context, product domain.Product) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{
				ProductID:   productID,
				ShopID:      "shop-1",
				OwnerUserID: "user-1",
				Status:      ProductStatusActive,
				Version:     4,
			}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1", Status: "active"}, nil
		},
	}, nil)

	_, err := service.Update(context.Background(), UpdateInput{
		ProductID:       "product-1",
		ShopID:          "shop-1",
		OwnerUserID:     "user-1",
		ExpectedVersion: 3,
		Name:            "Apple",
		Price:           10000,
		Currency:        "VND",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestDeleteProductMarksDeleted(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var saved domain.Product
	service := NewService(productRepositoryStub{
		save: func(ctx context.Context, product domain.Product) error {
			saved = product
			return nil
		},
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{
				ProductID:   productID,
				ShopID:      "shop-1",
				OwnerUserID: "user-1",
				Status:      ProductStatusActive,
				Version:     2,
			}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1", Status: "active"}, nil
		},
	}, auditLogger)

	product, err := service.Delete(context.Background(), DeleteInput{
		ProductID:       "product-1",
		ShopID:          "shop-1",
		OwnerUserID:     "user-1",
		ExpectedVersion: 2,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if product.Status != ProductStatusDeleted || saved.Status != ProductStatusDeleted {
		t.Fatalf("expected deleted product, got product=%#v saved=%#v", product, saved)
	}
	if product.Version != 3 || saved.Version != 3 {
		t.Fatalf("expected version 3, got product=%d saved=%d", product.Version, saved.Version)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}
