package firestore

import (
	"context"
	"fmt"
	"sort"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type ProductFreshnessReportRepository struct {
	client *gofirestore.Client
}

func NewProductFreshnessReportRepository(client *gofirestore.Client) *ProductFreshnessReportRepository {
	return &ProductFreshnessReportRepository{client: client}
}

func (r *ProductFreshnessReportRepository) Save(ctx context.Context, report domain.ProductFreshnessReport) error {
	_, err := r.client.Collection(ProductFreshnessReportsCollection).Doc(report.ReportID).Set(ctx, report)
	if err != nil {
		return fmt.Errorf("failed to save product freshness report: %w", err)
	}
	return nil
}

func (r *ProductFreshnessReportRepository) GetByID(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error) {
	doc, err := r.client.Collection(ProductFreshnessReportsCollection).Doc(reportID).Get(ctx)
	if err != nil {
		return domain.ProductFreshnessReport{}, fmt.Errorf("failed to get product freshness report: %w", err)
	}

	var report domain.ProductFreshnessReport
	if err := doc.DataTo(&report); err != nil {
		return domain.ProductFreshnessReport{}, fmt.Errorf("failed to decode product freshness report document: %w", err)
	}
	return report, nil
}

func (r *ProductFreshnessReportRepository) ListByProductID(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error) {
	docs, err := r.client.Collection(ProductFreshnessReportsCollection).Where("productId", "==", productID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list product freshness reports: %w", err)
	}

	return decodeFreshnessReports(docs)
}

func (r *ProductFreshnessReportRepository) ListByReporterUserID(ctx context.Context, reporterUserID string) ([]domain.ProductFreshnessReport, error) {
	docs, err := r.client.Collection(ProductFreshnessReportsCollection).Where("reporterUserId", "==", reporterUserID).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list product freshness reports by user: %w", err)
	}
	return decodeFreshnessReports(docs)
}

func decodeFreshnessReports(docs []*gofirestore.DocumentSnapshot) ([]domain.ProductFreshnessReport, error) {
	reports := make([]domain.ProductFreshnessReport, 0, len(docs))
	for _, doc := range docs {
		var report domain.ProductFreshnessReport
		if err := doc.DataTo(&report); err != nil {
			return nil, fmt.Errorf("failed to decode product freshness report document: %w", err)
		}
		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CreatedAt.After(reports[j].CreatedAt)
	})
	return reports, nil
}
