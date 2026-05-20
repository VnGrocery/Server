package batchbackfill

import (
	"context"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	batchsvc "vngrocery/internal/service/batch"
)

func TestRunCreatesDefaultBatchAndBackfillsLegacyRecords(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	products := productRepoStub{
		items: []domain.Product{{
			ProductID:      "product-1",
			ShopID:         "shop-1",
			OwnerUserID:    "seller-1",
			Category:       "fresh_produce",
			FreshnessScore: 8.5,
		}},
	}
	batches := &batchRepoStub{}
	pledges := &pledgeRepoStub{
		items: []domain.Pledge{
			{PledgeID: "pledge-1", ProductID: "product-1", ShopID: "shop-1"},
			{PledgeID: "pledge-2", ProductID: "product-1", ShopID: "shop-1", BatchID: "existing-batch"},
		},
	}
	reports := &reportRepoStub{
		items: []domain.ProductFreshnessReport{
			{ReportID: "report-1", ProductID: "product-1", ShopID: "shop-1"},
			{ReportID: "report-2", ProductID: "product-1", ShopID: "shop-1", BatchID: "existing-batch"},
		},
	}
	checks := &checkRepoStub{
		items: []domain.BuyerCheck{
			{CheckID: "check-1", ProductID: "product-1", ShopID: "shop-1"},
			{CheckID: "check-2", ProductID: "product-1", ShopID: "shop-1", BatchID: "existing-batch"},
		},
	}
	service := NewService(products, batches, pledges, reports, checks)
	service.now = func() time.Time { return now }

	result, err := service.Run(context.Background(), Options{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ProductsScanned != 1 || result.BatchesCreated != 1 || result.PledgesUpdated != 1 || result.ReportsUpdated != 1 || result.ChecksUpdated != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(batches.saved) != 1 {
		t.Fatalf("expected one saved batch, got %d", len(batches.saved))
	}
	batch := batches.saved[0]
	if batch.BatchID != "default-product-1" || batch.BatchCode != "DEFAULT-product-1" || batch.Status != batchsvc.StatusActive {
		t.Fatalf("unexpected batch: %#v", batch)
	}
	if batch.CurrentFreshness != 85 {
		t.Fatalf("expected normalized freshness 85, got %v", batch.CurrentFreshness)
	}
	if pledges.saved[0].BatchID != "default-product-1" || reports.saved[0].BatchID != "default-product-1" || checks.saved[0].BatchID != "default-product-1" {
		t.Fatalf("expected legacy records to get default batch")
	}
}

func TestRunDoesNotOverwriteExistingBatchOrSaveOnDryRun(t *testing.T) {
	products := productRepoStub{
		items: []domain.Product{{ProductID: "product-1", ShopID: "shop-1", FreshnessScore: 90}},
	}
	batches := &batchRepoStub{
		items: []domain.ProductBatch{{BatchID: "batch-1", ProductID: "product-1", ShopID: "shop-1"}},
	}
	pledges := &pledgeRepoStub{
		items: []domain.Pledge{{PledgeID: "pledge-1", ProductID: "product-1", ShopID: "shop-1"}},
	}
	reports := &reportRepoStub{
		items: []domain.ProductFreshnessReport{{ReportID: "report-1", ProductID: "product-1", ShopID: "shop-1"}},
	}
	checks := &checkRepoStub{
		items: []domain.BuyerCheck{{CheckID: "check-1", ProductID: "product-1", ShopID: "shop-1"}},
	}
	service := NewService(products, batches, pledges, reports, checks)

	result, err := service.Run(context.Background(), Options{DryRun: true})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.BatchesCreated != 0 || result.PledgesUpdated != 1 || result.ReportsUpdated != 1 || result.ChecksUpdated != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(batches.saved) != 0 || len(pledges.saved) != 0 || len(reports.saved) != 0 || len(checks.saved) != 0 {
		t.Fatal("dry-run should not save records")
	}
}

func TestRunHonorsStartAfterAndBatchSize(t *testing.T) {
	products := productRepoStub{
		items: []domain.Product{
			{ProductID: "product-1", ShopID: "shop-1"},
			{ProductID: "product-2", ShopID: "shop-1"},
			{ProductID: "product-3", ShopID: "shop-1"},
		},
	}
	service := NewService(products, &batchRepoStub{}, &pledgeRepoStub{}, &reportRepoStub{}, &checkRepoStub{})

	result, err := service.Run(context.Background(), Options{StartAfter: "product-1", BatchSize: 1})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ProductsScanned != 1 || result.BatchesCreated != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

type productRepoStub struct {
	items []domain.Product
}

func (s productRepoStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	return s.items, nil
}

type batchRepoStub struct {
	items []domain.ProductBatch
	saved []domain.ProductBatch
}

func (s *batchRepoStub) Save(ctx context.Context, batch domain.ProductBatch) error {
	s.saved = append(s.saved, batch)
	return nil
}

func (s *batchRepoStub) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	var out []domain.ProductBatch
	for _, batch := range s.items {
		if filter.ShopID != "" && batch.ShopID != filter.ShopID {
			continue
		}
		if filter.ProductID != "" && batch.ProductID != filter.ProductID {
			continue
		}
		out = append(out, batch)
	}
	return out, nil
}

type pledgeRepoStub struct {
	items []domain.Pledge
	saved []domain.Pledge
}

func (s *pledgeRepoStub) Save(ctx context.Context, pledge domain.Pledge) error {
	s.saved = append(s.saved, pledge)
	return nil
}

func (s *pledgeRepoStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	var out []domain.Pledge
	for _, pledge := range s.items {
		if pledge.ShopID == shopID {
			out = append(out, pledge)
		}
	}
	return out, nil
}

type reportRepoStub struct {
	items []domain.ProductFreshnessReport
	saved []domain.ProductFreshnessReport
}

func (s *reportRepoStub) Save(ctx context.Context, report domain.ProductFreshnessReport) error {
	s.saved = append(s.saved, report)
	return nil
}

func (s *reportRepoStub) List(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error) {
	var out []domain.ProductFreshnessReport
	for _, report := range s.items {
		if filter.ShopID != "" && report.ShopID != filter.ShopID {
			continue
		}
		if filter.ProductID != "" && report.ProductID != filter.ProductID {
			continue
		}
		out = append(out, report)
	}
	return out, nil
}

type checkRepoStub struct {
	items []domain.BuyerCheck
	saved []domain.BuyerCheck
}

func (s *checkRepoStub) Save(ctx context.Context, check domain.BuyerCheck) error {
	s.saved = append(s.saved, check)
	return nil
}

func (s *checkRepoStub) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	var out []domain.BuyerCheck
	for _, check := range s.items {
		if filter.ShopID != "" && check.ShopID != filter.ShopID {
			continue
		}
		if filter.ProductID != "" && check.ProductID != filter.ProductID {
			continue
		}
		out = append(out, check)
	}
	return out, nil
}
