package repository

import (
	"context"

	"vngrocery/internal/domain"
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, userID string) (domain.User, error)
}

type ShopRepository interface {
	Save(ctx context.Context, shop domain.Shop) error
	GetByID(ctx context.Context, shopID string) (domain.Shop, error)
}

type PledgeRepository interface {
	Save(ctx context.Context, pledge domain.Pledge) error
	GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error)
}
