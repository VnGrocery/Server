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
	ProductStatusActive  = "active"
	ProductStatusDeleted = "deleted"
)

type CreateInput struct {
	ShopID      string
	OwnerUserID string
	Name        string
	Description string
	Price       float64
	Currency    string
}

type UpdateInput struct {
	ProductID       string
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
	Name            string
	Description     string
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
	IncludeAllStatuses bool
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	products repository.ProductRepository
	shops    repository.ShopRepository
	audit    AuditLogger
	now      func() time.Time
}

func NewService(products repository.ProductRepository, shops repository.ShopRepository, auditLogger AuditLogger) *Service {
	return &Service{
		products: products,
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
		ProductID:   uuid.NewString(),
		ShopID:      shop.ShopID,
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Price:       input.Price,
		Currency:    normalizeCurrency(input.Currency),
		Status:      ProductStatusActive,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
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
	existing.Price = input.Price
	existing.Currency = normalizeCurrency(input.Currency)
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
	sort.Slice(products, func(i, j int) bool {
		return products[i].CreatedAt.Before(products[j].CreatedAt)
	})
	return products, nil
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
	if normalizeCurrency(currency) == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidProduct)
	}
	return nil
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}
