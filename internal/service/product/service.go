package product

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
	shopsvc "vngrocery/internal/service/shop"
)

var (
	ErrInvalidProduct  = errors.New("invalid product request")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("product not found")
	ErrVersionConflict = errors.New("version conflict")
)

const (
	ProductStatusActive           = "active"
	ProductStatusDraft            = "draft"
	ProductStatusPublished        = "published"
	ProductStatusArchived         = "archived"
	ProductStatusDeleted          = "deleted"
	FreshnessReportStatusActive   = "active"
	FreshnessReportStatusFlagged  = "flagged"
	FreshnessReportStatusRejected = "rejected"
	BatchStatusActive             = "active"
)

type CreateInput struct {
	ShopID         string
	OwnerUserID    string
	Name           string
	Description    string
	Category       string
	Tags           []string
	ImageURLs      []string
	FreshnessNote  string
	FreshnessScore float64
	Price          float64
	Currency       string
	Status         string
}

type UpdateInput struct {
	ProductID       string
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
	Name            string
	Description     string
	Category        string
	Tags            []string
	ImageURLs       []string
	FreshnessNote   string
	FreshnessScore  float64
	Price           float64
	Currency        string
	Status          string
}

type DeleteInput struct {
	ProductID       string
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
}

type ModerateInput struct {
	ProductID       string
	ModeratorUserID string
	ExpectedVersion int
	Status          string
	ModerationNote  string
}

type BulkUpsertInput struct {
	ShopID      string
	OwnerUserID string
	Items       []BulkUpsertItemInput
}

type BulkUpsertItemInput struct {
	ProductID       string
	ExpectedVersion int
	Name            string
	Description     string
	Category        string
	Tags            []string
	ImageURLs       []string
	FreshnessNote   string
	FreshnessScore  float64
	Price           float64
	Currency        string
	Status          string
}

type ListInput struct {
	ShopID             string
	OwnerUserID        string
	Query              string
	Category           string
	Tag                string
	Sort               string
	IncludeAllStatuses bool
}

type FreshnessReportInput struct {
	ProductID      string
	ShopID         string
	BatchID        string
	ReporterUserID string
	Score          float64
	Category       string
	Confidence     float64
	Comment        string
	ImageHash      string
	ImageCID       string
}

type ModerateFreshnessReportInput struct {
	ReportID        string
	ModeratorUserID string
	ExpectedVersion int
	Status          string
	ModerationNote  string
}

type ListFreshnessReportAdminInput struct {
	ActorUserID    string
	ReportID       string
	ShopID         string
	ProductID      string
	BatchID        string
	ReporterUserID string
	Status         string
	CreatedAfter   time.Time
	CreatedBefore  time.Time
	Page           int
	PageSize       int
}

type ListFreshnessReportAdminResult struct {
	Items    []domain.ProductFreshnessReport
	Page     int
	PageSize int
	Total    int
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	products repository.ProductRepository
	reports  repository.ProductFreshnessReportRepository
	batches  repository.ProductBatchRepository
	shops    repository.ShopRepository
	users    repository.UserRepository
	audit    AuditLogger
	now      func() time.Time
}

func NewService(products repository.ProductRepository, reports repository.ProductFreshnessReportRepository, shops repository.ShopRepository, users repository.UserRepository, auditLogger AuditLogger) *Service {
	return &Service{
		products: products,
		reports:  reports,
		shops:    shops,
		users:    users,
		audit:    auditLogger,
		now:      time.Now,
	}
}

func (s *Service) SetProductBatchRepository(batches repository.ProductBatchRepository) {
	s.batches = batches
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Product, error) {
	if err := validate(input.ShopID, input.OwnerUserID, input.Name, input.Price, input.Currency); err != nil {
		return domain.Product{}, err
	}
	status, err := validateProductStatus(input.Status)
	if err != nil {
		return domain.Product{}, err
	}
	if s.products == nil || s.shops == nil {
		return domain.Product{}, fmt.Errorf("product dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(input.ShopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	if shop.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Product{}, ErrForbidden
	}

	now := s.now().UTC()
	product := domain.Product{
		ProductID:      uuid.NewString(),
		ShopID:         shop.ShopID,
		OwnerUserID:    strings.TrimSpace(input.OwnerUserID),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(input.Description),
		Category:       strings.TrimSpace(input.Category),
		Tags:           normalizeStringSlice(input.Tags),
		ImageURLs:      normalizeStringSlice(input.ImageURLs),
		FreshnessNote:  strings.TrimSpace(input.FreshnessNote),
		FreshnessScore: input.FreshnessScore,
		Price:          input.Price,
		Currency:       defaultCurrency(input.Currency),
		Status:         status,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.products.Save(ctx, product); err != nil {
		return domain.Product{}, err
	}
	if err := s.logMutation(ctx, product.OwnerUserID, product.ProductID, product.Version, "product.created", audit.MutationPayload{After: product}, "created"); err != nil {
		return domain.Product{}, err
	}
	return product, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.Product, error) {
	if strings.TrimSpace(input.ProductID) == "" {
		return domain.Product{}, fmt.Errorf("%w: productId is required", ErrInvalidProduct)
	}
	if err := validate(input.ShopID, input.OwnerUserID, input.Name, input.Price, input.Currency); err != nil {
		return domain.Product{}, err
	}
	if input.ExpectedVersion <= 0 {
		return domain.Product{}, ErrVersionConflict
	}
	if s.products == nil || s.shops == nil {
		return domain.Product{}, fmt.Errorf("product dependencies are not configured")
	}
	if _, err := s.requireOwnedShop(ctx, input.ShopID, input.OwnerUserID); err != nil {
		return domain.Product{}, err
	}
	existing, err := s.products.GetByID(ctx, strings.TrimSpace(input.ProductID))
	if err != nil || existing.ProductID == "" || existing.Status == ProductStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	if existing.ShopID != strings.TrimSpace(input.ShopID) || existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Product{}, ErrForbidden
	}
	if existing.Version != input.ExpectedVersion {
		return domain.Product{}, ErrVersionConflict
	}

	before := existing
	existing.Name = strings.TrimSpace(input.Name)
	existing.Description = strings.TrimSpace(input.Description)
	existing.Category = strings.TrimSpace(input.Category)
	existing.Tags = normalizeStringSlice(input.Tags)
	existing.ImageURLs = normalizeStringSlice(input.ImageURLs)
	existing.FreshnessNote = strings.TrimSpace(input.FreshnessNote)
	existing.FreshnessScore = input.FreshnessScore
	existing.Price = input.Price
	existing.Currency = defaultCurrency(input.Currency)
	if strings.TrimSpace(input.Status) != "" {
		status, err := validateProductStatus(input.Status)
		if err != nil {
			return domain.Product{}, err
		}
		existing.Status = status
	}
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.products.Save(ctx, existing); err != nil {
		return domain.Product{}, err
	}
	if err := s.logMutation(ctx, existing.OwnerUserID, existing.ProductID, existing.Version, "product.updated", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "updated"); err != nil {
		return domain.Product{}, err
	}
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, input DeleteInput) (domain.Product, error) {
	if strings.TrimSpace(input.ProductID) == "" {
		return domain.Product{}, fmt.Errorf("%w: productId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Product{}, fmt.Errorf("%w: shopId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.OwnerUserID) == "" {
		return domain.Product{}, fmt.Errorf("%w: ownerUserId is required", ErrInvalidProduct)
	}
	if input.ExpectedVersion <= 0 {
		return domain.Product{}, ErrVersionConflict
	}
	if s.products == nil || s.shops == nil {
		return domain.Product{}, fmt.Errorf("product dependencies are not configured")
	}
	if _, err := s.requireOwnedShop(ctx, input.ShopID, input.OwnerUserID); err != nil {
		return domain.Product{}, err
	}
	existing, err := s.products.GetByID(ctx, strings.TrimSpace(input.ProductID))
	if err != nil || existing.ProductID == "" || existing.Status == ProductStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	if existing.ShopID != strings.TrimSpace(input.ShopID) || existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Product{}, ErrForbidden
	}
	if existing.Version != input.ExpectedVersion {
		return domain.Product{}, ErrVersionConflict
	}

	before := existing
	existing.Status = ProductStatusDeleted
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.products.Save(ctx, existing); err != nil {
		return domain.Product{}, err
	}
	if err := s.logMutation(ctx, existing.OwnerUserID, existing.ProductID, existing.Version, "product.deleted", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "deleted"); err != nil {
		return domain.Product{}, err
	}
	return existing, nil
}

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (domain.Product, error) {
	if strings.TrimSpace(input.ProductID) == "" {
		return domain.Product{}, fmt.Errorf("%w: productId is required", ErrInvalidProduct)
	}
	if input.ExpectedVersion <= 0 {
		return domain.Product{}, ErrVersionConflict
	}
	status, err := validateProductStatus(input.Status)
	if err != nil {
		return domain.Product{}, err
	}
	if s.products == nil || s.users == nil {
		return domain.Product{}, fmt.Errorf("product moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ModeratorUserID); err != nil {
		return domain.Product{}, err
	}
	existing, err := s.products.GetByID(ctx, strings.TrimSpace(input.ProductID))
	if err != nil || existing.ProductID == "" {
		return domain.Product{}, ErrNotFound
	}
	if existing.Version != input.ExpectedVersion {
		return domain.Product{}, ErrVersionConflict
	}

	before := existing
	now := s.now().UTC()
	existing.Status = status
	existing.ModeratedByUserID = strings.TrimSpace(input.ModeratorUserID)
	existing.ModerationNote = strings.TrimSpace(input.ModerationNote)
	existing.ModeratedAt = &now
	existing.Version++
	existing.UpdatedAt = now
	if err := s.products.Save(ctx, existing); err != nil {
		return domain.Product{}, err
	}
	if err := s.logMutation(ctx, existing.ModeratedByUserID, existing.ProductID, existing.Version, "product.moderated", audit.MutationPayload{Before: before, After: existing}, "moderated"); err != nil {
		return domain.Product{}, err
	}
	return existing, nil
}

func (s *Service) BulkUpsert(ctx context.Context, input BulkUpsertInput) ([]domain.Product, error) {
	if strings.TrimSpace(input.ShopID) == "" || strings.TrimSpace(input.OwnerUserID) == "" {
		return nil, fmt.Errorf("%w: shopId and ownerUserId are required", ErrInvalidProduct)
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("%w: items are required", ErrInvalidProduct)
	}
	results := make([]domain.Product, 0, len(input.Items))
	for _, item := range input.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			product, err := s.Create(ctx, CreateInput{
				ShopID:         input.ShopID,
				OwnerUserID:    input.OwnerUserID,
				Name:           item.Name,
				Description:    item.Description,
				Category:       item.Category,
				Tags:           item.Tags,
				ImageURLs:      item.ImageURLs,
				FreshnessNote:  item.FreshnessNote,
				FreshnessScore: item.FreshnessScore,
				Price:          item.Price,
				Currency:       item.Currency,
				Status:         item.Status,
			})
			if err != nil {
				return nil, err
			}
			results = append(results, product)
			continue
		}
		product, err := s.Update(ctx, UpdateInput{
			ProductID:       item.ProductID,
			ShopID:          input.ShopID,
			OwnerUserID:     input.OwnerUserID,
			ExpectedVersion: item.ExpectedVersion,
			Name:            item.Name,
			Description:     item.Description,
			Category:        item.Category,
			Tags:            item.Tags,
			ImageURLs:       item.ImageURLs,
			FreshnessNote:   item.FreshnessNote,
			FreshnessScore:  item.FreshnessScore,
			Price:           item.Price,
			Currency:        item.Currency,
			Status:          item.Status,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, product)
	}
	return results, nil
}

func (s *Service) GetByID(ctx context.Context, shopID, productID string) (domain.Product, error) {
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(productID) == "" {
		return domain.Product{}, fmt.Errorf("%w: shopId and productId are required", ErrInvalidProduct)
	}
	if s.products == nil || s.shops == nil {
		return domain.Product{}, fmt.Errorf("product dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	product, err := s.products.GetByID(ctx, strings.TrimSpace(productID))
	if err != nil || product.ProductID == "" || !isPublicProductStatus(product.Status) || product.ShopID != strings.TrimSpace(shopID) {
		return domain.Product{}, ErrNotFound
	}
	return product, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domain.Product, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return nil, fmt.Errorf("%w: shopId is required", ErrInvalidProduct)
	}
	if s.products == nil || s.shops == nil {
		return nil, fmt.Errorf("product dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(input.ShopID))
	if err != nil || shop.ShopID == "" || (!input.IncludeAllStatuses && shop.Status == shopsvc.ShopStatusDeleted) {
		return nil, ErrNotFound
	}
	status := ""
	if !input.IncludeAllStatuses {
		status = ProductStatusActive
	}
	products, err := s.products.List(ctx, repository.ProductListFilter{
		ShopID:      strings.TrimSpace(input.ShopID),
		Status:      status,
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
	})
	if err != nil {
		return nil, err
	}
	products = filterProducts(products, input)
	sortProducts(products, input.Sort)
	return products, nil
}

func (s *Service) ensureAdmin(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrForbidden
	}
	user, err := s.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return ErrForbidden
	}
	if !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CreateFreshnessReport(ctx context.Context, input FreshnessReportInput) (domain.ProductFreshnessReport, error) {
	if err := validateFreshnessReportInput(input); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	if s.products == nil || s.shops == nil || s.reports == nil {
		return domain.ProductFreshnessReport{}, fmt.Errorf("product freshness report dependencies are not configured")
	}
	if err := s.ensureFreshnessReportQuota(ctx, input.ReporterUserID); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	product, err := s.GetByID(ctx, input.ShopID, input.ProductID)
	if err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	batchID := strings.TrimSpace(input.BatchID)
	var batch domain.ProductBatch
	if batchID != "" {
		var err error
		batch, err = s.validateFreshnessReportBatch(ctx, product.ShopID, product.ProductID, batchID)
		if err != nil {
			return domain.ProductFreshnessReport{}, err
		}
	}

	now := s.now().UTC()
	report := domain.ProductFreshnessReport{
		ReportID:       uuid.NewString(),
		ProductID:      product.ProductID,
		ShopID:         product.ShopID,
		BatchID:        batchID,
		ReporterUserID: strings.TrimSpace(input.ReporterUserID),
		Status:         FreshnessReportStatusActive,
		Version:        1,
		Score:          input.Score,
		Category:       strings.TrimSpace(input.Category),
		Confidence:     input.Confidence,
		Comment:        strings.TrimSpace(input.Comment),
		ImageHash:      strings.TrimSpace(input.ImageHash),
		ImageCID:       strings.TrimSpace(input.ImageCID),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	if batchID != "" {
		if err := s.syncBatchFreshnessFromReport(ctx, batch, report); err != nil {
			return domain.ProductFreshnessReport{}, err
		}
	}
	if err := s.logReportMutation(ctx, report.ReporterUserID, report.ReportID, report.Version, "product_freshness_report.created", audit.MutationPayload{After: report}, "created"); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	return report, nil
}

func (s *Service) validateFreshnessReportBatch(ctx context.Context, shopID, productID, batchID string) (domain.ProductBatch, error) {
	if s.batches == nil {
		return domain.ProductBatch{}, fmt.Errorf("product batch repository is not configured")
	}
	batch, err := s.batches.GetByID(ctx, strings.TrimSpace(batchID))
	if err != nil || strings.TrimSpace(batch.BatchID) == "" {
		if err != nil {
			return domain.ProductBatch{}, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
		}
		return domain.ProductBatch{}, fmt.Errorf("%w: batchId not found", ErrInvalidProduct)
	}
	if batch.ShopID != strings.TrimSpace(shopID) {
		return domain.ProductBatch{}, fmt.Errorf("%w: batchId does not belong to shop", ErrInvalidProduct)
	}
	if batch.ProductID != strings.TrimSpace(productID) {
		return domain.ProductBatch{}, fmt.Errorf("%w: batchId does not belong to product", ErrInvalidProduct)
	}
	if strings.TrimSpace(batch.Status) != BatchStatusActive {
		return domain.ProductBatch{}, fmt.Errorf("%w: batch is not active", ErrInvalidProduct)
	}
	return batch, nil
}

func (s *Service) syncBatchFreshnessFromReport(ctx context.Context, batch domain.ProductBatch, report domain.ProductFreshnessReport) error {
	if s.batches == nil {
		return fmt.Errorf("product batch repository is not configured")
	}
	batch.CurrentFreshness = report.Score * 10
	batch.CurrentCategory = strings.TrimSpace(report.Category)
	batch.Version++
	batch.UpdatedAt = report.CreatedAt
	return s.batches.Save(ctx, batch)
}

func (s *Service) ModerateFreshnessReport(ctx context.Context, input ModerateFreshnessReportInput) (domain.ProductFreshnessReport, error) {
	if strings.TrimSpace(input.ReportID) == "" {
		return domain.ProductFreshnessReport{}, fmt.Errorf("%w: reportId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.ModeratorUserID) == "" {
		return domain.ProductFreshnessReport{}, fmt.Errorf("%w: moderatorUserId is required", ErrInvalidProduct)
	}
	if input.ExpectedVersion <= 0 {
		return domain.ProductFreshnessReport{}, ErrVersionConflict
	}
	if s.reports == nil || s.users == nil {
		return domain.ProductFreshnessReport{}, fmt.Errorf("product freshness report moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ModeratorUserID); err != nil {
		return domain.ProductFreshnessReport{}, err
	}

	report, err := s.reports.GetByID(ctx, strings.TrimSpace(input.ReportID))
	if err != nil || report.ReportID == "" {
		return domain.ProductFreshnessReport{}, ErrNotFound
	}
	if report.Version != input.ExpectedVersion {
		return domain.ProductFreshnessReport{}, ErrVersionConflict
	}
	status, err := validateFreshnessReportStatus(input.Status)
	if err != nil {
		return domain.ProductFreshnessReport{}, err
	}

	before := report
	report.Status = status
	report.Version++
	report.ModeratedByUserID = strings.TrimSpace(input.ModeratorUserID)
	report.ModerationNote = strings.TrimSpace(input.ModerationNote)
	now := s.now().UTC()
	report.ModeratedAt = &now
	report.UpdatedAt = now
	if err := s.reports.Save(ctx, report); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	if err := s.logReportMutation(ctx, input.ModeratorUserID, report.ReportID, report.Version, "product_freshness_report.moderated", audit.MutationPayload{
		Before: before,
		After:  report,
	}, report.Status); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	return report, nil
}

func (s *Service) ListFreshnessReports(ctx context.Context, shopID, productID, batchID string) ([]domain.ProductFreshnessReport, error) {
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(productID) == "" {
		return nil, fmt.Errorf("%w: shopId and productId are required", ErrInvalidProduct)
	}
	if s.products == nil || s.shops == nil || s.reports == nil {
		return nil, fmt.Errorf("product freshness report dependencies are not configured")
	}
	if _, err := s.GetByID(ctx, shopID, productID); err != nil {
		return nil, err
	}
	reports, err := s.reports.List(ctx, repository.ProductFreshnessReportListFilter{
		ShopID:    strings.TrimSpace(shopID),
		ProductID: strings.TrimSpace(productID),
		BatchID:   strings.TrimSpace(batchID),
		Status:    FreshnessReportStatusActive,
	})
	if err != nil {
		return nil, err
	}
	active := make([]domain.ProductFreshnessReport, 0, len(reports))
	for _, report := range reports {
		if report.ShopID == strings.TrimSpace(shopID) && report.ProductID == strings.TrimSpace(productID) && report.Status == FreshnessReportStatusActive {
			active = append(active, report)
		}
	}
	return active, nil
}

func (s *Service) ListFreshnessReportsAdmin(ctx context.Context, input ListFreshnessReportAdminInput) (ListFreshnessReportAdminResult, error) {
	if strings.TrimSpace(input.ActorUserID) == "" {
		return ListFreshnessReportAdminResult{}, fmt.Errorf("%w: actorUserId is required", ErrInvalidProduct)
	}
	if s.reports == nil || s.users == nil {
		return ListFreshnessReportAdminResult{}, fmt.Errorf("product freshness report list dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return ListFreshnessReportAdminResult{}, err
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	items, err := s.reports.List(ctx, repository.ProductFreshnessReportListFilter{
		ReportID:       strings.TrimSpace(input.ReportID),
		ShopID:         strings.TrimSpace(input.ShopID),
		ProductID:      strings.TrimSpace(input.ProductID),
		BatchID:        strings.TrimSpace(input.BatchID),
		ReporterUserID: strings.TrimSpace(input.ReporterUserID),
		Status:         strings.TrimSpace(input.Status),
		CreatedAfter:   input.CreatedAfter,
		CreatedBefore:  input.CreatedBefore,
	})
	if err != nil {
		return ListFreshnessReportAdminResult{}, err
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return ListFreshnessReportAdminResult{Items: []domain.ProductFreshnessReport{}, Page: page, PageSize: pageSize, Total: total}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return ListFreshnessReportAdminResult{
		Items:    items[start:end],
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *Service) requireOwnedShop(ctx context.Context, shopID, ownerUserID string) (domain.Shop, error) {
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.Shop{}, ErrNotFound
	}
	if shop.OwnerUserID != strings.TrimSpace(ownerUserID) {
		return domain.Shop{}, ErrForbidden
	}
	return shop, nil
}

func (s *Service) logMutation(ctx context.Context, actorUserID, productID string, version int, action string, payload any, status string) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     strings.TrimSpace(actorUserID),
		ResourceType:    "product",
		ResourceID:      strings.TrimSpace(productID),
		ResourceVersion: version,
		Action:          action,
		Status:          status,
		Payload:         payload,
	})
}

func (s *Service) logReportMutation(ctx context.Context, actorUserID, reportID string, version int, action string, payload any, status string) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     strings.TrimSpace(actorUserID),
		ResourceType:    "product_freshness_report",
		ResourceID:      strings.TrimSpace(reportID),
		ResourceVersion: version,
		Action:          action,
		Status:          status,
		Payload:         payload,
	})
}

func validate(shopID, ownerUserID, name string, price float64, currency string) error {
	if strings.TrimSpace(shopID) == "" {
		return fmt.Errorf("%w: shopId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(ownerUserID) == "" {
		return fmt.Errorf("%w: ownerUserId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidProduct)
	}
	if price < 0 {
		return fmt.Errorf("%w: price must be greater than or equal to 0", ErrInvalidProduct)
	}
	if price > 0 && normalizeCurrency(currency) == "" {
		return fmt.Errorf("%w: currency is required when price is set", ErrInvalidProduct)
	}
	return nil
}

func validateProductStatus(status string) (string, error) {
	normalized := normalizeProductStatus(status)
	switch normalized {
	case ProductStatusActive, ProductStatusDraft, ProductStatusPublished, ProductStatusArchived, ProductStatusDeleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: invalid status", ErrInvalidProduct)
	}
}

func normalizeProductStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return ProductStatusActive
	}
	return normalized
}

func isPublicProductStatus(status string) bool {
	return status == ProductStatusActive || status == ProductStatusPublished
}

func validateFreshnessReportInput(input FreshnessReportInput) error {
	if strings.TrimSpace(input.ShopID) == "" {
		return fmt.Errorf("%w: shopId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.ProductID) == "" {
		return fmt.Errorf("%w: productId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.ReporterUserID) == "" {
		return fmt.Errorf("%w: reporterUserId is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.Category) == "" {
		return fmt.Errorf("%w: category is required", ErrInvalidProduct)
	}
	if strings.TrimSpace(input.ImageHash) == "" {
		return fmt.Errorf("%w: imageHash is required", ErrInvalidProduct)
	}
	if input.Score < 0 || input.Score > 10 {
		return fmt.Errorf("%w: score must be between 0 and 10", ErrInvalidProduct)
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be between 0 and 1", ErrInvalidProduct)
	}
	return nil
}

func validateFreshnessReportStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case FreshnessReportStatusActive, FreshnessReportStatusFlagged, FreshnessReportStatusRejected:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: invalid freshness report status", ErrInvalidProduct)
	}
}

func (s *Service) ensureFreshnessReportQuota(ctx context.Context, reporterUserID string) error {
	reports, err := s.reports.ListByReporterUserID(ctx, strings.TrimSpace(reporterUserID))
	if err != nil {
		return err
	}
	since := s.now().UTC().Add(-1 * time.Hour)
	count := 0
	for _, report := range reports {
		if report.CreatedAt.After(since) {
			count++
		}
	}
	if count >= 10 {
		return fmt.Errorf("%w: freshness report rate limit exceeded", ErrInvalidProduct)
	}
	return nil
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func defaultCurrency(currency string) string {
	normalized := normalizeCurrency(currency)
	if normalized == "" {
		return "VND"
	}
	return normalized
}

func normalizeStringSlice(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func filterProducts(products []domain.Product, input ListInput) []domain.Product {
	query := strings.ToLower(strings.TrimSpace(input.Query))
	category := strings.ToLower(strings.TrimSpace(input.Category))
	tag := strings.ToLower(strings.TrimSpace(input.Tag))
	if query == "" && category == "" && tag == "" {
		return products
	}
	filtered := make([]domain.Product, 0, len(products))
	for _, product := range products {
		if category != "" && strings.ToLower(strings.TrimSpace(product.Category)) != category {
			continue
		}
		if tag != "" && !hasTag(product.Tags, tag) {
			continue
		}
		if query != "" && !matchesProductQuery(product, query) {
			continue
		}
		filtered = append(filtered, product)
	}
	return filtered
}

func hasTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if strings.ToLower(strings.TrimSpace(tag)) == expected {
			return true
		}
	}
	return false
}

func matchesProductQuery(product domain.Product, query string) bool {
	if strings.Contains(strings.ToLower(product.Name), query) || strings.Contains(strings.ToLower(product.Description), query) || strings.Contains(strings.ToLower(product.Category), query) {
		return true
	}
	for _, tag := range product.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func sortProducts(products []domain.Product, sortBy string) {
	switch strings.TrimSpace(sortBy) {
	case "name":
		sort.Slice(products, func(i, j int) bool { return strings.ToLower(products[i].Name) < strings.ToLower(products[j].Name) })
	case "freshnessScore":
		sort.Slice(products, func(i, j int) bool { return products[i].FreshnessScore > products[j].FreshnessScore })
	default:
		sort.Slice(products, func(i, j int) bool { return products[i].CreatedAt.Before(products[j].CreatedAt) })
	}
}
