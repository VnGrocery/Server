package batchbackfill

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	batchsvc "vngrocery/internal/service/batch"
)

type ProductRepository interface {
	List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error)
}

type ProductBatchRepository interface {
	Save(ctx context.Context, batch domain.ProductBatch) error
	List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error)
}

type PledgeRepository interface {
	Save(ctx context.Context, pledge domain.Pledge) error
	ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error)
}

type ProductFreshnessReportRepository interface {
	Save(ctx context.Context, report domain.ProductFreshnessReport) error
	List(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error)
}

type BuyerCheckRepository interface {
	Save(ctx context.Context, check domain.BuyerCheck) error
	List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error)
}

type Options struct {
	DryRun        bool
	StartAfter    string
	BatchSize     int
	DefaultStatus string
}

type Result struct {
	ProductsScanned int
	BatchesCreated  int
	PledgesUpdated  int
	ReportsUpdated  int
	ChecksUpdated   int
}

type Service struct {
	products ProductRepository
	batches  ProductBatchRepository
	pledges  PledgeRepository
	reports  ProductFreshnessReportRepository
	checks   BuyerCheckRepository
	now      func() time.Time
}

func NewService(products ProductRepository, batches ProductBatchRepository, pledges PledgeRepository, reports ProductFreshnessReportRepository, checks BuyerCheckRepository) *Service {
	return &Service{
		products: products,
		batches:  batches,
		pledges:  pledges,
		reports:  reports,
		checks:   checks,
		now:      time.Now,
	}
}

func (s *Service) Run(ctx context.Context, options Options) (Result, error) {
	if s.products == nil || s.batches == nil || s.pledges == nil || s.reports == nil || s.checks == nil {
		return Result{}, fmt.Errorf("batch backfill dependencies are not configured")
	}
	status := strings.TrimSpace(options.DefaultStatus)
	if status == "" {
		status = batchsvc.StatusActive
	}
	products, err := s.products.List(ctx, repository.ProductListFilter{})
	if err != nil {
		return Result{}, err
	}

	startAfter := strings.TrimSpace(options.StartAfter)
	result := Result{}
	for _, product := range products {
		productID := strings.TrimSpace(product.ProductID)
		if productID == "" {
			continue
		}
		if startAfter != "" && productID <= startAfter {
			continue
		}
		if options.BatchSize > 0 && result.ProductsScanned >= options.BatchSize {
			break
		}
		result.ProductsScanned++

		batch, created, err := s.ensureDefaultBatch(ctx, product, status, options.DryRun)
		if err != nil {
			return result, err
		}
		if created {
			result.BatchesCreated++
		}
		batchID := strings.TrimSpace(batch.BatchID)
		if batchID == "" {
			continue
		}

		pledgesUpdated, err := s.backfillPledges(ctx, product, batchID, options.DryRun)
		if err != nil {
			return result, err
		}
		result.PledgesUpdated += pledgesUpdated

		reportsUpdated, err := s.backfillReports(ctx, product, batchID, options.DryRun)
		if err != nil {
			return result, err
		}
		result.ReportsUpdated += reportsUpdated

		checksUpdated, err := s.backfillChecks(ctx, product, batchID, options.DryRun)
		if err != nil {
			return result, err
		}
		result.ChecksUpdated += checksUpdated
	}

	return result, nil
}

func (s *Service) ensureDefaultBatch(ctx context.Context, product domain.Product, status string, dryRun bool) (domain.ProductBatch, bool, error) {
	existing, err := s.batches.List(ctx, repository.ProductBatchListFilter{
		ShopID:    strings.TrimSpace(product.ShopID),
		ProductID: strings.TrimSpace(product.ProductID),
	})
	if err != nil {
		return domain.ProductBatch{}, false, err
	}
	if len(existing) > 0 {
		return existing[0], false, nil
	}

	now := s.now().UTC()
	batch := domain.ProductBatch{
		BatchID:          defaultBatchID(product.ProductID),
		ProductID:        strings.TrimSpace(product.ProductID),
		ShopID:           strings.TrimSpace(product.ShopID),
		OwnerUserID:      strings.TrimSpace(product.OwnerUserID),
		BatchCode:        "DEFAULT-" + strings.TrimSpace(product.ProductID),
		CurrentFreshness: normalizeFreshnessPercent(product.FreshnessScore),
		CurrentCategory:  strings.TrimSpace(product.Category),
		Status:           status,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if dryRun {
		return batch, true, nil
	}
	if err := s.batches.Save(ctx, batch); err != nil {
		return domain.ProductBatch{}, false, err
	}
	return batch, true, nil
}

func (s *Service) backfillPledges(ctx context.Context, product domain.Product, batchID string, dryRun bool) (int, error) {
	pledges, err := s.pledges.ListByShopID(ctx, strings.TrimSpace(product.ShopID))
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, pledge := range pledges {
		if strings.TrimSpace(pledge.ProductID) != strings.TrimSpace(product.ProductID) || strings.TrimSpace(pledge.BatchID) != "" {
			continue
		}
		pledge.BatchID = batchID
		pledge.UpdatedAt = s.now().UTC()
		updated++
		if dryRun {
			continue
		}
		if err := s.pledges.Save(ctx, pledge); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func (s *Service) backfillReports(ctx context.Context, product domain.Product, batchID string, dryRun bool) (int, error) {
	reports, err := s.reports.List(ctx, repository.ProductFreshnessReportListFilter{
		ShopID:    strings.TrimSpace(product.ShopID),
		ProductID: strings.TrimSpace(product.ProductID),
	})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, report := range reports {
		if strings.TrimSpace(report.BatchID) != "" {
			continue
		}
		report.BatchID = batchID
		report.UpdatedAt = s.now().UTC()
		updated++
		if dryRun {
			continue
		}
		if err := s.reports.Save(ctx, report); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func (s *Service) backfillChecks(ctx context.Context, product domain.Product, batchID string, dryRun bool) (int, error) {
	checks, err := s.checks.List(ctx, repository.BuyerCheckListFilter{
		ShopID:    strings.TrimSpace(product.ShopID),
		ProductID: strings.TrimSpace(product.ProductID),
	})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, check := range checks {
		if strings.TrimSpace(check.BatchID) != "" {
			continue
		}
		check.BatchID = batchID
		check.UpdatedAt = s.now().UTC()
		updated++
		if dryRun {
			continue
		}
		if err := s.checks.Save(ctx, check); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func defaultBatchID(productID string) string {
	return "default-" + strings.TrimSpace(productID)
}

func normalizeFreshnessPercent(score float64) float64 {
	switch {
	case score < 0:
		return 0
	case score <= 10:
		return score * 10
	case score > 100:
		return 100
	default:
		return score
	}
}
