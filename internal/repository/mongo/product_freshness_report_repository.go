package mongo

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
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
func (r *ProductFreshnessReportRepository) listSorted(ctx context.Context, filter bson.M) ([]domain.ProductFreshnessReport, error) {
	items, err := listDocuments[domain.ProductFreshnessReport](ctx, r.collection, filter)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
