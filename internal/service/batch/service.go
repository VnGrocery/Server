package batch

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
	productsvc "vngrocery/internal/service/product"
	shopsvc "vngrocery/internal/service/shop"
)

var (
	ErrInvalidBatch    = errors.New("invalid product batch request")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("product batch not found")
	ErrVersionConflict = errors.New("version conflict")
)

const (
	StatusActive   = "active"
	StatusSoldOut  = "sold_out"
	StatusExpired  = "expired"
	StatusRecalled = "recalled"
	StatusDeleted  = "deleted"
)

type CreateInput struct {
	ShopID           string
	ProductID        string
	OwnerUserID      string
	BatchCode        string
	OriginName       string
	OriginAddress    string
	SupplierName     string
	SlaughteredAt    *time.Time
	PackedAt         *time.Time
	ReceivedAt       *time.Time
	ExpiresAt        *time.Time
	Quantity         float64
	QuantityUnit     string
	StorageTempMin   float64
	StorageTempMax   float64
	CurrentFreshness float64
	CurrentCategory  string
	Status           string
}

type UpdateInput struct {
	BatchID          string
	ShopID           string
	ProductID        string
	OwnerUserID      string
	ExpectedVersion  int
	BatchCode        string
	OriginName       string
	OriginAddress    string
	SupplierName     string
	SlaughteredAt    *time.Time
	PackedAt         *time.Time
	ReceivedAt       *time.Time
	ExpiresAt        *time.Time
	Quantity         float64
	QuantityUnit     string
	StorageTempMin   float64
	StorageTempMax   float64
	CurrentFreshness float64
	CurrentCategory  string
	Status           string
}

type DeleteInput struct {
	BatchID         string
	ShopID          string
	ProductID       string
	OwnerUserID     string
	ExpectedVersion int
}

type ListInput struct {
	ShopID             string
	ProductID          string
	OwnerUserID        string
	Status             string
	IncludeAllStatuses bool
}

type Service struct {
	batches  repository.ProductBatchRepository
	products repository.ProductRepository
	shops    repository.ShopRepository
	now      func() time.Time
}

func NewService(batches repository.ProductBatchRepository, products repository.ProductRepository, shops repository.ShopRepository) *Service {
	return &Service{
		batches:  batches,
		products: products,
		shops:    shops,
		now:      time.Now,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.ProductBatch, error) {
	if err := validateCreateInput(input); err != nil {
		return domain.ProductBatch{}, err
	}
	if s.batches == nil || s.products == nil || s.shops == nil {
		return domain.ProductBatch{}, fmt.Errorf("product batch dependencies are not configured")
	}
	product, err := s.requireOwnedProduct(ctx, input.ShopID, input.ProductID, input.OwnerUserID)
	if err != nil {
		return domain.ProductBatch{}, err
	}
	status, err := validateStatus(input.Status)
	if err != nil {
		return domain.ProductBatch{}, err
	}

	now := s.now().UTC()
	batch := domain.ProductBatch{
		BatchID:          uuid.NewString(),
		ProductID:        product.ProductID,
		ShopID:           product.ShopID,
		OwnerUserID:      strings.TrimSpace(input.OwnerUserID),
		BatchCode:        strings.TrimSpace(input.BatchCode),
		OriginName:       strings.TrimSpace(input.OriginName),
		OriginAddress:    strings.TrimSpace(input.OriginAddress),
		SupplierName:     strings.TrimSpace(input.SupplierName),
		SlaughteredAt:    input.SlaughteredAt,
		PackedAt:         input.PackedAt,
		ReceivedAt:       input.ReceivedAt,
		ExpiresAt:        input.ExpiresAt,
		Quantity:         input.Quantity,
		QuantityUnit:     defaultQuantityUnit(input.QuantityUnit),
		StorageTempMin:   input.StorageTempMin,
		StorageTempMax:   input.StorageTempMax,
		CurrentFreshness: normalizeFreshnessPercent(input.CurrentFreshness),
		CurrentCategory:  strings.TrimSpace(input.CurrentCategory),
		Status:           status,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.batches.Save(ctx, batch); err != nil {
		return domain.ProductBatch{}, err
	}
	return batch, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.ProductBatch, error) {
	if strings.TrimSpace(input.BatchID) == "" {
		return domain.ProductBatch{}, fmt.Errorf("%w: batchId is required", ErrInvalidBatch)
	}
	if input.ExpectedVersion <= 0 {
		return domain.ProductBatch{}, ErrVersionConflict
	}
	if err := validateMutableFields(input.BatchCode, input.Quantity, input.StorageTempMin, input.StorageTempMax, input.CurrentFreshness); err != nil {
		return domain.ProductBatch{}, err
	}
	if s.batches == nil || s.products == nil || s.shops == nil {
		return domain.ProductBatch{}, fmt.Errorf("product batch dependencies are not configured")
	}
	product, err := s.requireOwnedProduct(ctx, input.ShopID, input.ProductID, input.OwnerUserID)
	if err != nil {
		return domain.ProductBatch{}, err
	}
	existing, err := s.batches.GetByID(ctx, strings.TrimSpace(input.BatchID))
	if err != nil || existing.BatchID == "" || existing.Status == StatusDeleted {
		return domain.ProductBatch{}, ErrNotFound
	}
	if existing.ShopID != product.ShopID || existing.ProductID != product.ProductID || existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.ProductBatch{}, ErrForbidden
	}
	if existing.Version != input.ExpectedVersion {
		return domain.ProductBatch{}, ErrVersionConflict
	}
	status, err := validateStatus(input.Status)
	if err != nil {
		return domain.ProductBatch{}, err
	}

	existing.BatchCode = strings.TrimSpace(input.BatchCode)
	existing.OriginName = strings.TrimSpace(input.OriginName)
	existing.OriginAddress = strings.TrimSpace(input.OriginAddress)
	existing.SupplierName = strings.TrimSpace(input.SupplierName)
	existing.SlaughteredAt = input.SlaughteredAt
	existing.PackedAt = input.PackedAt
	existing.ReceivedAt = input.ReceivedAt
	existing.ExpiresAt = input.ExpiresAt
	existing.Quantity = input.Quantity
	existing.QuantityUnit = defaultQuantityUnit(input.QuantityUnit)
	existing.StorageTempMin = input.StorageTempMin
	existing.StorageTempMax = input.StorageTempMax
	existing.CurrentFreshness = normalizeFreshnessPercent(input.CurrentFreshness)
	existing.CurrentCategory = strings.TrimSpace(input.CurrentCategory)
	existing.Status = status
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.batches.Save(ctx, existing); err != nil {
		return domain.ProductBatch{}, err
	}
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, input DeleteInput) (domain.ProductBatch, error) {
	if strings.TrimSpace(input.BatchID) == "" {
		return domain.ProductBatch{}, fmt.Errorf("%w: batchId is required", ErrInvalidBatch)
	}
	if input.ExpectedVersion <= 0 {
		return domain.ProductBatch{}, ErrVersionConflict
	}
	if s.batches == nil || s.products == nil || s.shops == nil {
		return domain.ProductBatch{}, fmt.Errorf("product batch dependencies are not configured")
	}
	product, err := s.requireOwnedProduct(ctx, input.ShopID, input.ProductID, input.OwnerUserID)
	if err != nil {
		return domain.ProductBatch{}, err
	}
	existing, err := s.batches.GetByID(ctx, strings.TrimSpace(input.BatchID))
	if err != nil || existing.BatchID == "" || existing.Status == StatusDeleted {
		return domain.ProductBatch{}, ErrNotFound
	}
	if existing.ShopID != product.ShopID || existing.ProductID != product.ProductID || existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.ProductBatch{}, ErrForbidden
	}
	if existing.Version != input.ExpectedVersion {
		return domain.ProductBatch{}, ErrVersionConflict
	}
	existing.Status = StatusDeleted
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.batches.Save(ctx, existing); err != nil {
		return domain.ProductBatch{}, err
	}
	return existing, nil
}

func (s *Service) GetByID(ctx context.Context, shopID, productID, batchID string) (domain.ProductBatch, error) {
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(productID) == "" || strings.TrimSpace(batchID) == "" {
		return domain.ProductBatch{}, fmt.Errorf("%w: shopId, productId and batchId are required", ErrInvalidBatch)
	}
	if s.batches == nil || s.products == nil || s.shops == nil {
		return domain.ProductBatch{}, fmt.Errorf("product batch dependencies are not configured")
	}
	if _, err := s.requirePublicProduct(ctx, shopID, productID); err != nil {
		return domain.ProductBatch{}, err
	}
	batch, err := s.batches.GetByID(ctx, strings.TrimSpace(batchID))
	if err != nil || batch.BatchID == "" || batch.Status == StatusDeleted {
		return domain.ProductBatch{}, ErrNotFound
	}
	if batch.ShopID != strings.TrimSpace(shopID) || batch.ProductID != strings.TrimSpace(productID) {
		return domain.ProductBatch{}, ErrNotFound
	}
	return batch, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domain.ProductBatch, error) {
	if strings.TrimSpace(input.ShopID) == "" || strings.TrimSpace(input.ProductID) == "" {
		return nil, fmt.Errorf("%w: shopId and productId are required", ErrInvalidBatch)
	}
	if s.batches == nil || s.products == nil || s.shops == nil {
		return nil, fmt.Errorf("product batch dependencies are not configured")
	}
	if input.IncludeAllStatuses {
		if _, err := s.requireOwnedProduct(ctx, input.ShopID, input.ProductID, input.OwnerUserID); err != nil {
			return nil, err
		}
	} else if _, err := s.requirePublicProduct(ctx, input.ShopID, input.ProductID); err != nil {
		return nil, err
	}

	status := strings.TrimSpace(input.Status)
	if status == "" && !input.IncludeAllStatuses {
		status = StatusActive
	}
	if status != "" {
		validated, err := validateStatus(status)
		if err != nil {
			return nil, err
		}
		status = validated
	}
	batches, err := s.batches.List(ctx, repository.ProductBatchListFilter{
		ShopID:      strings.TrimSpace(input.ShopID),
		ProductID:   strings.TrimSpace(input.ProductID),
		Status:      status,
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
	})
	if err != nil {
		return nil, err
	}
	if !input.IncludeAllStatuses {
		filtered := batches[:0]
		for _, batch := range batches {
			if batch.Status != StatusDeleted {
				filtered = append(filtered, batch)
			}
		}
		batches = filtered
	}
	sort.SliceStable(batches, func(i, j int) bool {
		return batches[i].UpdatedAt.After(batches[j].UpdatedAt)
	})
	return batches, nil
}

func (s *Service) requireOwnedProduct(ctx context.Context, shopID, productID, ownerUserID string) (domain.Product, error) {
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(productID) == "" || strings.TrimSpace(ownerUserID) == "" {
		return domain.Product{}, fmt.Errorf("%w: shopId, productId and ownerUserId are required", ErrInvalidBatch)
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	if shop.OwnerUserID != strings.TrimSpace(ownerUserID) {
		return domain.Product{}, ErrForbidden
	}
	product, err := s.products.GetByID(ctx, strings.TrimSpace(productID))
	if err != nil || product.ProductID == "" || product.Status == productsvc.ProductStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	if product.ShopID != shop.ShopID || product.OwnerUserID != strings.TrimSpace(ownerUserID) {
		return domain.Product{}, ErrForbidden
	}
	return product, nil
}

func (s *Service) requirePublicProduct(ctx context.Context, shopID, productID string) (domain.Product, error) {
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil || shop.ShopID == "" || shop.Status == shopsvc.ShopStatusDeleted {
		return domain.Product{}, ErrNotFound
	}
	product, err := s.products.GetByID(ctx, strings.TrimSpace(productID))
	if err != nil || product.ProductID == "" || product.ShopID != strings.TrimSpace(shopID) || product.Status != productsvc.ProductStatusActive {
		return domain.Product{}, ErrNotFound
	}
	return product, nil
}

func validateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.ShopID) == "" || strings.TrimSpace(input.ProductID) == "" || strings.TrimSpace(input.OwnerUserID) == "" {
		return fmt.Errorf("%w: shopId, productId and ownerUserId are required", ErrInvalidBatch)
	}
	return validateMutableFields(input.BatchCode, input.Quantity, input.StorageTempMin, input.StorageTempMax, input.CurrentFreshness)
}

func validateMutableFields(batchCode string, quantity, tempMin, tempMax, freshness float64) error {
	if strings.TrimSpace(batchCode) == "" {
		return fmt.Errorf("%w: batchCode is required", ErrInvalidBatch)
	}
	if quantity < 0 {
		return fmt.Errorf("%w: quantity must be non-negative", ErrInvalidBatch)
	}
	if tempMin > tempMax {
		return fmt.Errorf("%w: storageTempMin must be less than or equal to storageTempMax", ErrInvalidBatch)
	}
	if _, err := NormalizeFreshnessPercent(freshness); err != nil {
		return err
	}
	return nil
}

func NormalizeFreshnessPercent(score float64) (float64, error) {
	if score < 0 {
		return 0, fmt.Errorf("%w: currentFreshness must be between 0 and 100 percent", ErrInvalidBatch)
	}
	if score <= 10 {
		return score * 10, nil
	}
	if score <= 100 {
		return score, nil
	}
	return 0, fmt.Errorf("%w: currentFreshness must be between 0 and 100 percent", ErrInvalidBatch)
}

func normalizeFreshnessPercent(score float64) float64 {
	normalized, err := NormalizeFreshnessPercent(score)
	if err != nil {
		return score
	}
	return normalized
}

func validateStatus(status string) (string, error) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return StatusActive, nil
	}
	switch normalized {
	case StatusActive, StatusSoldOut, StatusExpired, StatusRecalled, StatusDeleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: unsupported status %q", ErrInvalidBatch, status)
	}
}

func defaultQuantityUnit(unit string) string {
	normalized := strings.TrimSpace(unit)
	if normalized == "" {
		return "kg"
	}
	return normalized
}
