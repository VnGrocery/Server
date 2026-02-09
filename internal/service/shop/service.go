package shop

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

var (
	ErrInvalidShop = errors.New("invalid shop request")
	ErrForbidden   = errors.New("forbidden")
	ErrNotFound    = errors.New("shop not found")
)

const ShopStatusActive = "active"

type CreateInput struct {
	OwnerUserID string
	Name        string
	Description string
	Address     string
	Latitude    float64
	Longitude   float64
}

type UpdateInput struct {
	ShopID      string
	OwnerUserID string
	Name        string
	Description string
	Address     string
	Latitude    float64
	Longitude   float64
}

type Service struct {
	shops repository.ShopRepository
	now   func() time.Time
}

func NewService(shops repository.ShopRepository) *Service {
	return &Service{
		shops: shops,
		now:   time.Now,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Shop, error) {
	if err := validate(input.OwnerUserID, input.Name, input.Address, input.Latitude, input.Longitude); err != nil {
		return domain.Shop{}, err
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}

	now := s.now().UTC()
	shop := domain.Shop{
		ShopID:      uuid.NewString(),
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Address:     strings.TrimSpace(input.Address),
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Status:      ShopStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.shops.Save(ctx, shop); err != nil {
		return domain.Shop{}, err
	}

	return shop, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.Shop, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if err := validate(input.OwnerUserID, input.Name, input.Address, input.Latitude, input.Longitude); err != nil {
		return domain.Shop{}, err
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}

	existing, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if existing.ShopID == "" {
		return domain.Shop{}, ErrNotFound
	}
	if existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Shop{}, ErrForbidden
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Description = strings.TrimSpace(input.Description)
	existing.Address = strings.TrimSpace(input.Address)
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude
	existing.UpdatedAt = s.now().UTC()

	if err := s.shops.Save(ctx, existing); err != nil {
		return domain.Shop{}, err
	}

	return existing, nil
}

func (s *Service) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	if strings.TrimSpace(shopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return shop, nil
}

func (s *Service) ListActive(ctx context.Context) ([]domain.Shop, error) {
	if s.shops == nil {
		return nil, fmt.Errorf("shop repository is not configured")
	}
	return s.shops.ListActive(ctx)
}

func validate(ownerUserID, name, address string, latitude, longitude float64) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return fmt.Errorf("%w: ownerUserId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidShop)
	}
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("%w: address is required", ErrInvalidShop)
	}
	if math.IsNaN(latitude) || latitude < -90 || latitude > 90 {
		return fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalidShop)
	}
	if math.IsNaN(longitude) || longitude < -180 || longitude > 180 {
		return fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalidShop)
	}
	return nil
}
