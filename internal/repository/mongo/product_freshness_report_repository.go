package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductFreshnessReportRepository struct{ collection *mongo.Collection }

func NewProductFreshnessReportRepository(db *mongo.Database) *ProductFreshnessReportRepository {
	return &ProductFreshnessReportRepository{collection: db.Collection(productFreshnessReportsCollection)}
}
func (r *ProductFreshnessReportRepository) Save(ctx context.Context, report domain.ProductFreshnessReport) error {
	return saveByID(ctx, r.collection, report.ReportID, report)
}
func (r *ProductFreshnessReportRepository) GetByID(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error) {
	return getByID[domain.ProductFreshnessReport](ctx, r.collection, reportID)
}
func (r *ProductFreshnessReportRepository) ListByProductID(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error) {
	return r.listSorted(ctx, bson.M{"productId": productID})
}
func (r *ProductFreshnessReportRepository) ListByReporterUserID(ctx context.Context, reporterUserID string) ([]domain.ProductFreshnessReport, error) {
	return r.listSorted(ctx, bson.M{"reporterUserId": reporterUserID})
}
func (r *ProductFreshnessReportRepository) List(ctx context.Context, filter repository.ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error) {
	query := bson.M{}
	if filter.ReportID != "" {
		query["reportId"] = filter.ReportID
	}
	if filter.ShopID != "" {
		query["shopId"] = filter.ShopID
	}
	if filter.ProductID != "" {
		query["productId"] = filter.ProductID
	}
	if filter.ReporterUserID != "" {
		query["reporterUserId"] = filter.ReporterUserID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if !filter.CreatedAfter.IsZero() || !filter.CreatedBefore.IsZero() {
		rangeFilter := bson.M{}
		if !filter.CreatedAfter.IsZero() {
			rangeFilter["$gte"] = filter.CreatedAfter
		}
		if !filter.CreatedBefore.IsZero() {
			rangeFilter["$lte"] = filter.CreatedBefore
		}
		query["createdAt"] = rangeFilter
	}
	return r.listSorted(ctx, query)
}
func (r *ProductFreshnessReportRepository) listSorted(ctx context.Context, filter bson.M) ([]domain.ProductFreshnessReport, error) {
	items, err := listDocuments[domain.ProductFreshnessReport](ctx, r.collection, filter)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
