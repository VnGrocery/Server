package batch

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	productsvc "vngrocery/internal/service/product"
	shopsvc "vngrocery/internal/service/shop"
)

type batchRepositoryStub struct {
	save    func(ctx context.Context, batch domain.ProductBatch) error
	getByID func(ctx context.Context, batchID string) (domain.ProductBatch, error)
	list    func(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error)
}

func (s batchRepositoryStub) Save(ctx context.Context, batch domain.ProductBatch) error {
	if s.save != nil {
		return s.save(ctx, batch)
	}
	return nil
}

func (s batchRepositoryStub) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	if s.getByID != nil {
		return s.getByID(ctx, batchID)
	}
	return domain.ProductBatch{}, nil
}

func (s batchRepositoryStub) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, nil
}

type productRepositoryStub struct {
	getByID func(ctx context.Context, productID string) (domain.Product, error)
}

func (s productRepositoryStub) Save(ctx context.Context, product domain.Product) error { return nil }
func (s productRepositoryStub) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	if s.getByID != nil {
		return s.getByID(ctx, productID)
	}
	return domain.Product{}, nil
}
func (s productRepositoryStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
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

func TestCreateBatch(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	var saved domain.ProductBatch
	service := NewService(batchRepositoryStub{
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			saved = batch
			return nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-1", Status: productsvc.ProductStatusActive}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
		},
	})
	service.now = func() time.Time { return now }

	batch, err := service.Create(context.Background(), CreateInput{
		ShopID:           "shop-1",
		ProductID:        "product-1",
		OwnerUserID:      "seller-1",
		BatchCode:        "BATCH-001",
		Quantity:         12.5,
		CurrentFreshness: 92,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if batch.BatchID == "" || saved.BatchID != batch.BatchID {
		t.Fatalf("expected generated and saved batch id, got batch=%#v saved=%#v", batch, saved)
	}
	if batch.Status != StatusActive || batch.QuantityUnit != "kg" || batch.Version != 1 {
		t.Fatalf("unexpected batch defaults: %#v", batch)
	}
}

func TestCreateBatchNormalizesFreshnessScoreToPercent(t *testing.T) {
	service := newOwnedBatchService(t, batchRepositoryStub{
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			if batch.CurrentFreshness != 85 {
				t.Fatalf("expected normalized freshness 85, got %v", batch.CurrentFreshness)
			}
			return nil
		},
	})

	batch, err := service.Create(context.Background(), CreateInput{
		ShopID:           "shop-1",
		ProductID:        "product-1",
		OwnerUserID:      "seller-1",
		BatchCode:        "BATCH-001",
		CurrentFreshness: 8.5,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if batch.CurrentFreshness != 85 {
		t.Fatalf("expected normalized freshness in response, got %v", batch.CurrentFreshness)
	}
}

func TestCreateBatchRejectsFreshnessOutsidePercentScale(t *testing.T) {
	service := newOwnedBatchService(t, batchRepositoryStub{
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.Create(context.Background(), CreateInput{
		ShopID:           "shop-1",
		ProductID:        "product-1",
		OwnerUserID:      "seller-1",
		BatchCode:        "BATCH-001",
		CurrentFreshness: 101,
	})
	if !errors.Is(err, ErrInvalidBatch) {
		t.Fatalf("expected ErrInvalidBatch, got %v", err)
	}
}

func TestCreateBatchRejectsForeignShop(t *testing.T) {
	service := NewService(batchRepositoryStub{
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			t.Fatal("save should not be called")
			return nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-2", Status: productsvc.ProductStatusActive}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
		},
	})

	_, err := service.Create(context.Background(), CreateInput{
		ShopID:           "shop-1",
		ProductID:        "product-1",
		OwnerUserID:      "seller-1",
		BatchCode:        "BATCH-001",
		CurrentFreshness: 90,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func newOwnedBatchService(t *testing.T, batches batchRepositoryStub) *Service {
	t.Helper()
	return NewService(batches, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-1", Status: productsvc.ProductStatusActive}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
		},
	})
}

func TestUpdateBatchRejectsVersionConflict(t *testing.T) {
	service := NewService(batchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-1", OwnerUserID: "seller-1", Status: StatusActive, Version: 3}, nil
		},
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			t.Fatal("save should not be called")
			return nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-1", Status: productsvc.ProductStatusActive}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
		},
	})

	_, err := service.Update(context.Background(), UpdateInput{
		BatchID:          "batch-1",
		ShopID:           "shop-1",
		ProductID:        "product-1",
		OwnerUserID:      "seller-1",
		ExpectedVersion:  2,
		BatchCode:        "BATCH-001",
		CurrentFreshness: 88,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestListBatchesDefaultsToActive(t *testing.T) {
	service := NewService(batchRepositoryStub{
		list: func(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
			if filter.ShopID != "shop-1" || filter.ProductID != "product-1" || filter.Status != StatusActive {
				t.Fatalf("unexpected list filter: %+v", filter)
			}
			return []domain.ProductBatch{{BatchID: "batch-1", Status: StatusActive}}, nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", OwnerUserID: "seller-1", Status: productsvc.ProductStatusActive}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1", Status: shopsvc.ShopStatusActive}, nil
		},
	})

	batches, err := service.List(context.Background(), ListInput{ShopID: "shop-1", ProductID: "product-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(batches) != 1 || batches[0].BatchID != "batch-1" {
		t.Fatalf("unexpected batches: %#v", batches)
	}
}
