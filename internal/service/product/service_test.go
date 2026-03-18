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

type productFreshnessReportRepositoryStub struct {
	save            func(ctx context.Context, report domain.ProductFreshnessReport) error
	listByProductID func(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error)
}

func (s productFreshnessReportRepositoryStub) Save(ctx context.Context, report domain.ProductFreshnessReport) error {
	if s.save != nil {
		return s.save(ctx, report)
	}
	return nil
}

func (s productFreshnessReportRepositoryStub) ListByProductID(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error) {
	if s.listByProductID != nil {
		return s.listByProductID(ctx, productID)
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
	}, productFreshnessReportRepositoryStub{}, shopRepositoryStub{
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
	}, productFreshnessReportRepositoryStub{}, shopRepositoryStub{
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
	}, productFreshnessReportRepositoryStub{}, shopRepositoryStub{
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

func TestCreateFreshnessReport(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	service := NewService(productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", Status: ProductStatusActive}, nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			if report.ProductID != "product-1" || report.ShopID != "shop-1" || report.ReporterUserID != "buyer-1" || report.Status != FreshnessReportStatusActive || report.Version != 1 {
				t.Fatalf("unexpected report: %#v", report)
			}
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: "active"}, nil
		},
	}, auditLogger)

	report, err := service.CreateFreshnessReport(context.Background(), FreshnessReportInput{
		ProductID:      "product-1",
		ShopID:         "shop-1",
		ReporterUserID: "buyer-1",
		Score:          6.2,
		Category:       "bruised_fruit",
		Confidence:     0.86,
		Comment:        "Bruised on one side",
		ImageHash:      "hash-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.ReportID == "" {
		t.Fatal("expected report id")
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestListFreshnessReportsFiltersActiveByProduct(t *testing.T) {
	service := NewService(productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", Status: ProductStatusActive}, nil
		},
	}, productFreshnessReportRepositoryStub{
		listByProductID: func(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error) {
			return []domain.ProductFreshnessReport{
				{ReportID: "report-1", ProductID: productID, ShopID: "shop-1", Status: FreshnessReportStatusActive},
				{ReportID: "report-2", ProductID: productID, ShopID: "shop-1", Status: "deleted"},
			}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: "active"}, nil
		},
	}, nil)

	reports, err := service.ListFreshnessReports(context.Background(), "shop-1", "product-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(reports) != 1 || reports[0].ReportID != "report-1" {
		t.Fatalf("unexpected reports: %#v", reports)
	}
}
