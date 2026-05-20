package product

import (
	"context"
	"errors"
	"testing"
	"time"

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
	save             func(ctx context.Context, report domain.ProductFreshnessReport) error
	getByID          func(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error)
	listByProductID  func(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error)
	listByReporterID func(ctx context.Context, reporterUserID string) ([]domain.ProductFreshnessReport, error)
	list             func(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error)
}

func (s productFreshnessReportRepositoryStub) Save(ctx context.Context, report domain.ProductFreshnessReport) error {
	if s.save != nil {
		return s.save(ctx, report)
	}
	return nil
}

func (s productFreshnessReportRepositoryStub) GetByID(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error) {
	if s.getByID != nil {
		return s.getByID(ctx, reportID)
	}
	return domain.ProductFreshnessReport{}, nil
}

func (s productFreshnessReportRepositoryStub) ListByProductID(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error) {
	if s.listByProductID != nil {
		return s.listByProductID(ctx, productID)
	}
	return nil, nil
}

func (s productFreshnessReportRepositoryStub) ListByReporterUserID(ctx context.Context, reporterUserID string) ([]domain.ProductFreshnessReport, error) {
	if s.listByReporterID != nil {
		return s.listByReporterID(ctx, reporterUserID)
	}
	return nil, nil
}

func (s productFreshnessReportRepositoryStub) List(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, nil
}

type productBatchRepositoryStub struct {
	save    func(ctx context.Context, batch domain.ProductBatch) error
	getByID func(ctx context.Context, batchID string) (domain.ProductBatch, error)
	list    func(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error)
}

func (s productBatchRepositoryStub) Save(ctx context.Context, batch domain.ProductBatch) error {
	if s.save != nil {
		return s.save(ctx, batch)
	}
	return nil
}

func (s productBatchRepositoryStub) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	if s.getByID != nil {
		return s.getByID(ctx, batchID)
	}
	return domain.ProductBatch{}, nil
}

func (s productBatchRepositoryStub) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
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

type userRepositoryStub struct {
	getByID func(ctx context.Context, userID string) (domain.User, error)
}

func (s userRepositoryStub) Save(ctx context.Context, user domain.User) error { return nil }
func (s userRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.User{}, nil
}
func (s userRepositoryStub) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
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
	}, userRepositoryStub{}, auditLogger)

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
	}, userRepositoryStub{}, nil)

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
	}, userRepositoryStub{}, auditLogger)

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
			if report.ProductID != "product-1" || report.ShopID != "shop-1" || report.BatchID != "batch-1" || report.ReporterUserID != "buyer-1" || report.Status != FreshnessReportStatusActive || report.Version != 1 {
				t.Fatalf("unexpected report: %#v", report)
			}
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: "active"}, nil
		},
	}, userRepositoryStub{}, auditLogger)
	service.SetProductBatchRepository(productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-1", Status: BatchStatusActive}, nil
		},
	})

	report, err := service.CreateFreshnessReport(context.Background(), FreshnessReportInput{
		ProductID:      "product-1",
		ShopID:         "shop-1",
		BatchID:        "batch-1",
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

func TestCreateFreshnessReportSyncsBatchFreshness(t *testing.T) {
	now := time.Date(2026, 5, 21, 9, 30, 0, 0, time.UTC)
	var savedBatch domain.ProductBatch
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{
				BatchID:          batchID,
				ShopID:           "shop-1",
				ProductID:        "product-1",
				OwnerUserID:      "seller-1",
				BatchCode:        "BATCH-001",
				CurrentFreshness: 91,
				CurrentCategory:  "fresh_produce",
				Status:           BatchStatusActive,
				Version:          3,
				CreatedAt:        now.Add(-24 * time.Hour),
				UpdatedAt:        now.Add(-24 * time.Hour),
			}, nil
		},
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			savedBatch = batch
			return nil
		},
	}, productFreshnessReportRepositoryStub{})
	service.now = func() time.Time { return now }

	report, err := service.CreateFreshnessReport(context.Background(), FreshnessReportInput{
		ProductID:      "product-1",
		ShopID:         "shop-1",
		BatchID:        "batch-1",
		ReporterUserID: "buyer-1",
		Score:          8.2,
		Category:       "good_storage",
		Confidence:     0.86,
		ImageHash:      "hash-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.ReportID == "" {
		t.Fatal("expected report id")
	}
	if savedBatch.BatchID != "batch-1" {
		t.Fatalf("expected saved batch, got %#v", savedBatch)
	}
	if savedBatch.CurrentFreshness != 82 {
		t.Fatalf("expected current freshness 82, got %v", savedBatch.CurrentFreshness)
	}
	if savedBatch.CurrentCategory != "good_storage" {
		t.Fatalf("expected current category from report, got %q", savedBatch.CurrentCategory)
	}
	if savedBatch.Version != 4 {
		t.Fatalf("expected batch version 4, got %d", savedBatch.Version)
	}
	if !savedBatch.UpdatedAt.Equal(now) {
		t.Fatalf("expected batch updatedAt to report time, got %v", savedBatch.UpdatedAt)
	}
	if savedBatch.BatchCode != "BATCH-001" || savedBatch.OwnerUserID != "seller-1" {
		t.Fatalf("expected existing batch fields preserved, got %#v", savedBatch)
	}
}

func TestCreateFreshnessReportReturnsErrorWhenBatchFreshnessSyncFails(t *testing.T) {
	reportSaved := false
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-1", Status: BatchStatusActive, Version: 1}, nil
		},
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			return errors.New("save batch failed")
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			reportSaved = true
			return nil
		},
	})

	_, err := service.CreateFreshnessReport(context.Background(), validFreshnessReportInput())
	if err == nil {
		t.Fatal("expected error")
	}
	if !reportSaved {
		t.Fatal("expected report to be saved before batch sync failure")
	}
}

func TestCreateFreshnessReportRejectsMissingBatch(t *testing.T) {
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{}, nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.CreateFreshnessReport(context.Background(), validFreshnessReportInput())
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCreateFreshnessReportRejectsBatchFromDifferentProduct(t *testing.T) {
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-2", Status: BatchStatusActive}, nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.CreateFreshnessReport(context.Background(), validFreshnessReportInput())
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCreateFreshnessReportRejectsBatchFromDifferentShop(t *testing.T) {
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-2", ProductID: "product-1", Status: BatchStatusActive}, nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.CreateFreshnessReport(context.Background(), validFreshnessReportInput())
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCreateFreshnessReportRejectsInactiveBatch(t *testing.T) {
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{BatchID: batchID, ShopID: "shop-1", ProductID: "product-1", Status: "recalled"}, nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			t.Fatal("save should not be called")
			return nil
		},
	})

	_, err := service.CreateFreshnessReport(context.Background(), validFreshnessReportInput())
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCreateFreshnessReportAllowsLegacyReportWithoutBatch(t *testing.T) {
	service := newFreshnessReportService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			t.Fatal("batch lookup should not be called")
			return domain.ProductBatch{}, nil
		},
		save: func(ctx context.Context, batch domain.ProductBatch) error {
			t.Fatal("batch save should not be called")
			return nil
		},
	}, productFreshnessReportRepositoryStub{
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			if report.BatchID != "" {
				t.Fatalf("expected empty batch id, got %q", report.BatchID)
			}
			return nil
		},
	})
	input := validFreshnessReportInput()
	input.BatchID = ""

	report, err := service.CreateFreshnessReport(context.Background(), input)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.BatchID != "" {
		t.Fatalf("expected empty batch id, got %q", report.BatchID)
	}
}

func TestListFreshnessReportsFiltersActiveByProduct(t *testing.T) {
	service := NewService(productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", Status: ProductStatusActive}, nil
		},
	}, productFreshnessReportRepositoryStub{
		list: func(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error) {
			if filter.ProductID != "product-1" || filter.BatchID != "batch-1" || filter.Status != FreshnessReportStatusActive {
				t.Fatalf("unexpected filter: %+v", filter)
			}
			return []domain.ProductFreshnessReport{
				{ReportID: "report-1", ProductID: filter.ProductID, ShopID: "shop-1", BatchID: filter.BatchID, Status: FreshnessReportStatusActive},
				{ReportID: "report-2", ProductID: filter.ProductID, ShopID: "shop-1", BatchID: filter.BatchID, Status: "deleted"},
			}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: "active"}, nil
		},
	}, userRepositoryStub{}, nil)

	reports, err := service.ListFreshnessReports(context.Background(), "shop-1", "product-1", "batch-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(reports) != 1 || reports[0].ReportID != "report-1" {
		t.Fatalf("unexpected reports: %#v", reports)
	}
}

func newFreshnessReportService(t *testing.T, batches productBatchRepositoryStub, reports productFreshnessReportRepositoryStub) *Service {
	t.Helper()
	service := NewService(productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1", Status: ProductStatusActive}, nil
		},
	}, reports, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: "active"}, nil
		},
	}, userRepositoryStub{}, nil)
	service.SetProductBatchRepository(batches)
	return service
}

func validFreshnessReportInput() FreshnessReportInput {
	return FreshnessReportInput{
		ProductID:      "product-1",
		ShopID:         "shop-1",
		BatchID:        "batch-1",
		ReporterUserID: "buyer-1",
		Score:          6.2,
		Category:       "bruised_fruit",
		Confidence:     0.86,
		ImageHash:      "hash-1",
	}
}

func TestModerateFreshnessReport(t *testing.T) {
	service := NewService(productRepositoryStub{}, productFreshnessReportRepositoryStub{
		getByID: func(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error) {
			return domain.ProductFreshnessReport{ReportID: reportID, Status: FreshnessReportStatusActive, Version: 1}, nil
		},
		save: func(ctx context.Context, report domain.ProductFreshnessReport) error {
			if report.Status != FreshnessReportStatusFlagged || report.Version != 2 || report.ModeratedByUserID != "admin-1" {
				t.Fatalf("unexpected moderated report: %#v", report)
			}
			return nil
		},
	}, shopRepositoryStub{}, userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "admin"}, nil
		},
	}, &auditLoggerStub{})

	report, err := service.ModerateFreshnessReport(context.Background(), ModerateFreshnessReportInput{
		ReportID:        "report-1",
		ModeratorUserID: "admin-1",
		ExpectedVersion: 1,
		Status:          FreshnessReportStatusFlagged,
		ModerationNote:  "suspicious",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.Status != FreshnessReportStatusFlagged || report.Version != 2 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestListFreshnessReportsAdmin(t *testing.T) {
	service := NewService(productRepositoryStub{}, productFreshnessReportRepositoryStub{
		list: func(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error) {
			if filter.ShopID != "shop-1" || filter.Status != FreshnessReportStatusActive {
				t.Fatalf("unexpected filter: %#v", filter)
			}
			return []domain.ProductFreshnessReport{
				{ReportID: "report-1", Status: FreshnessReportStatusActive},
				{ReportID: "report-2", Status: FreshnessReportStatusActive},
			}, nil
		},
	}, shopRepositoryStub{}, userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "admin"}, nil
		},
	}, nil)

	result, err := service.ListFreshnessReportsAdmin(context.Background(), ListFreshnessReportAdminInput{
		ActorUserID: "admin-1",
		ShopID:      "shop-1",
		Status:      FreshnessReportStatusActive,
		Page:        1,
		PageSize:    1,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].ReportID != "report-1" {
		t.Fatalf("unexpected list result: %#v", result)
	}
}
