package shop

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

var (
	ErrInvalidShop   = errors.New("invalid shop request")
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("shop not found")
	ErrAdminRequired = errors.New("admin role is required")
)

const (
	ShopStatusActive        = "active"
	ShopStatusPendingReview = "pending_review"
	ShopStatusSuspended     = "suspended"
	ShopStatusRejected      = "rejected"
	defaultPage             = 1
	defaultPageSize         = 20
	maxPageSize             = 100
)

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

type ModerateInput struct {
	ShopID          string
	ModeratorUserID string
	Status          string
	ModerationNote  string
}

type ListInput struct {
	Page               int
	PageSize           int
	Query              string
	Status             string
	OwnerUserID        string
	ActorUserID        string
	IncludeAllStatuses bool
}

type TrustSummary struct {
	HasPledges         bool
	PledgeCount        int
	LatestPledgeID     string
	LatestPledgeStatus string
	LatestScore        float64
	LatestCategory     string
	LatestConfidence   float64
	LastCommittedAt    *time.Time
}

type ShopView struct {
	Shop         domain.Shop
	TrustSummary TrustSummary
}

type ListResult struct {
	Items    []ShopView
	Page     int
	PageSize int
	Total    int
	HasNext  bool
}

type Service struct {
	shops   repository.ShopRepository
	pledges repository.PledgeRepository
	users   repository.UserRepository
	now     func() time.Time
}

func NewService(shops repository.ShopRepository, pledges repository.PledgeRepository, users repository.UserRepository) *Service {
	return &Service{
		shops:   shops,
		pledges: pledges,
		users:   users,
		now:     time.Now,
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

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (domain.Shop, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.ModeratorUserID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: moderatorUserId is required", ErrInvalidShop)
	}
	status := strings.TrimSpace(input.Status)
	if !isAllowedModerationStatus(status) {
		return domain.Shop{}, fmt.Errorf("%w: unsupported moderation status", ErrInvalidShop)
	}
	if s.shops == nil || s.users == nil {
		return domain.Shop{}, fmt.Errorf("shop moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ModeratorUserID); err != nil {
		return domain.Shop{}, err
	}

	existing, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	now := s.now().UTC()
	existing.Status = status
	existing.ModeratedByUserID = strings.TrimSpace(input.ModeratorUserID)
	existing.ModerationNote = strings.TrimSpace(input.ModerationNote)
	existing.ModeratedAt = &now
	existing.UpdatedAt = now

	if err := s.shops.Save(ctx, existing); err != nil {
		return domain.Shop{}, err
	}

	return existing, nil
}

func (s *Service) GetByID(ctx context.Context, shopID string) (ShopView, error) {
	if strings.TrimSpace(shopID) == "" {
		return ShopView{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if s.shops == nil {
		return ShopView{}, fmt.Errorf("shop repository is not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil {
		return ShopView{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	return s.buildShopView(ctx, shop)
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, error) {
	if s.shops == nil {
		return ListResult{}, fmt.Errorf("shop repository is not configured")
	}

	page, pageSize := normalizePagination(input.Page, input.PageSize)
	filter := repository.ShopListFilter{
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
	}
	if input.IncludeAllStatuses {
		if strings.TrimSpace(input.ActorUserID) == "" {
			return ListResult{}, ErrAdminRequired
		}
		if s.users == nil {
			return ListResult{}, fmt.Errorf("user repository is not configured")
		}
		if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
			return ListResult{}, err
		}
		filter.Status = strings.TrimSpace(input.Status)
	} else {
		status := strings.TrimSpace(input.Status)
		if status == "" {
			filter.Status = ShopStatusActive
		} else {
			if status != ShopStatusActive {
				return ListResult{}, fmt.Errorf("%w: public shop listing only supports active status", ErrInvalidShop)
			}
			filter.Status = status
		}
	}

	shops, err := s.shops.List(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	filtered := make([]domain.Shop, 0, len(shops))
	for _, shop := range shops {
		if query == "" || matchesQuery(shop, query) {
			filtered = append(filtered, shop)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})

	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return ListResult{
			Items:    []ShopView{},
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			HasNext:  false,
		}, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	pageItems := filtered[start:end]
	items := make([]ShopView, 0, len(pageItems))
	for _, shop := range pageItems {
		view, err := s.buildShopView(ctx, shop)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, view)
	}

	return ListResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		HasNext:  end < total,
	}, nil
}

func (s *Service) buildShopView(ctx context.Context, shop domain.Shop) (ShopView, error) {
	shopView := ShopView{Shop: shop}
	if s.pledges == nil {
		return shopView, nil
	}

	pledges, err := s.pledges.ListByShopID(ctx, shop.ShopID)
	if err != nil {
		return ShopView{}, err
	}
	if len(pledges) == 0 {
		return shopView, nil
	}

	latest := pledges[0]
	shopView.TrustSummary = TrustSummary{
		HasPledges:         true,
		PledgeCount:        len(pledges),
		LatestPledgeID:     latest.PledgeID,
		LatestPledgeStatus: latest.Status,
		LatestScore:        latest.Score,
		LatestCategory:     latest.Category,
		LatestConfidence:   latest.Confidence,
		LastCommittedAt:    &latest.CreatedAt,
	}
	return shopView, nil
}

func (s *Service) ensureAdmin(ctx context.Context, userID string) error {
	user, err := s.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdminRequired, err)
	}
	if !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		return ErrAdminRequired
	}
	return nil
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func matchesQuery(shop domain.Shop, query string) bool {
	name := strings.ToLower(shop.Name)
	address := strings.ToLower(shop.Address)
	description := strings.ToLower(shop.Description)
	return strings.Contains(name, query) || strings.Contains(address, query) || strings.Contains(description, query)
}

func isAllowedModerationStatus(status string) bool {
	switch status {
	case ShopStatusActive, ShopStatusPendingReview, ShopStatusSuspended, ShopStatusRejected:
		return true
	default:
		return false
	}
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
