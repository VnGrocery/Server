package repository

import (
	"context"

	"vngrocery/internal/domain"
)

type ShopListFilter struct {
	Status      string
	OwnerUserID string
}

type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, userID string) (domain.User, error)
}

type ShopRepository interface {
	Save(ctx context.Context, shop domain.Shop) error
	GetByID(ctx context.Context, shopID string) (domain.Shop, error)
	List(ctx context.Context, filter ShopListFilter) ([]domain.Shop, error)
}

type PledgeRepository interface {
	Save(ctx context.Context, pledge domain.Pledge) error
	GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error)
}

type ShopReviewRepository interface {
	Save(ctx context.Context, review domain.ShopReview) error
	GetByShopAndUser(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.ShopReview, error)
}

type AuthUserRepository interface {
	NewUserID() string
	Save(ctx context.Context, user domain.AuthUser) error
	GetByID(ctx context.Context, userID string) (domain.AuthUser, error)
	GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error)
	GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error)
}

type EventLogRepository interface {
	Save(ctx context.Context, event domain.EventLog) error
}
