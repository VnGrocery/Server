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
	ProductStatusActive         = "active"
	ProductStatusDeleted        = "deleted"
	FreshnessReportStatusActive = "active"
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
}

type DeleteInput struct {
	ProductID       string
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
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
	ReporterUserID string
	Score          float64
	Category       string
	Confidence     float64
	Comment        string
	ImageHash      string
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	products repository.ProductRepository
	reports  repository.ProductFreshnessReportRepository
	shops    repository.ShopRepository
	audit    AuditLogger
	now      func() time.Time
}

func NewService(products repository.ProductRepository, reports repository.ProductFreshnessReportRepository, shops repository.ShopRepository, auditLogger AuditLogger) *Service {
	return &Service{
		products: products,
		reports:  reports,
		shops:    shops,
		audit:    auditLogger,
		now:      time.Now,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Product, error) {
	if err := validate(input.ShopID, input.OwnerUserID, input.Name, input.Price, input.Currency); err != nil {
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
		Status:         ProductStatusActive,
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
	if err != nil || product.ProductID == "" || product.Status != ProductStatusActive || product.ShopID != strings.TrimSpace(shopID) {
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

func (s *Service) CreateFreshnessReport(ctx context.Context, input FreshnessReportInput) (domain.ProductFreshnessReport, error) {
	if err := validateFreshnessReportInput(input); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	if s.products == nil || s.shops == nil || s.reports == nil {
		return domain.ProductFreshnessReport{}, fmt.Errorf("product freshness report dependencies are not configured")
	}
	product, err := s.GetByID(ctx, input.ShopID, input.ProductID)
	if err != nil {
		return domain.ProductFreshnessReport{}, err
	}

	now := s.now().UTC()
	report := domain.ProductFreshnessReport{
		ReportID:       uuid.NewString(),
		ProductID:      product.ProductID,
		ShopID:         product.ShopID,
		ReporterUserID: strings.TrimSpace(input.ReporterUserID),
		Status:         FreshnessReportStatusActive,
		Version:        1,
		Score:          input.Score,
		Category:       strings.TrimSpace(input.Category),
		Confidence:     input.Confidence,
		Comment:        strings.TrimSpace(input.Comment),
		ImageHash:      strings.TrimSpace(input.ImageHash),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.reports.Save(ctx, report); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	if err := s.logReportMutation(ctx, report.ReporterUserID, report.ReportID, report.Version, "product_freshness_report.created", audit.MutationPayload{After: report}, "created"); err != nil {
		return domain.ProductFreshnessReport{}, err
	}
	return report, nil
}

func (s *Service) ListFreshnessReports(ctx context.Context, shopID, productID string) ([]domain.ProductFreshnessReport, error) {
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(productID) == "" {
		return nil, fmt.Errorf("%w: shopId and productId are required", ErrInvalidProduct)
	}
	if s.products == nil || s.shops == nil || s.reports == nil {
		return nil, fmt.Errorf("product freshness report dependencies are not configured")
	}
	if _, err := s.GetByID(ctx, shopID, productID); err != nil {
		return nil, err
	}
	reports, err := s.reports.ListByProductID(ctx, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	active := make([]domain.ProductFreshnessReport, 0, len(reports))
	for _, report := range reports {
		if report.ShopID == strings.TrimSpace(shopID) && report.Status == FreshnessReportStatusActive {
			active = append(active, report)
		}
	}
	return active, nil
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
