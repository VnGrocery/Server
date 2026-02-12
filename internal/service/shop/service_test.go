package shop

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type shopRepositoryStub struct {
	save    func(ctx context.Context, shop domain.Shop) error
	getByID func(ctx context.Context, shopID string) (domain.Shop, error)
	list    func(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error)
}

func (s shopRepositoryStub) Save(ctx context.Context, shop domain.Shop) error {
	return s.save(ctx, shop)
}

func (s shopRepositoryStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return s.getByID(ctx, shopID)
}

func (s shopRepositoryStub) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	if s.list == nil {
		return nil, errors.New("not implemented")
	}
	return s.list(ctx, filter)
}

type pledgeRepositoryStub struct {
	listByShopID func(ctx context.Context, shopID string) ([]domain.Pledge, error)
}

func (p pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error { return nil }
func (p pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	return domain.Pledge{}, nil
}
func (p pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	if p.listByShopID == nil {
		return nil, nil
	}
	return p.listByShopID(ctx, shopID)
}

type userRepositoryStub struct {
	getByID func(ctx context.Context, userID string) (domain.User, error)
}

func (u userRepositoryStub) Save(ctx context.Context, user domain.User) error { return nil }
func (u userRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	if u.getByID == nil {
		return domain.User{}, errors.New("not implemented")
	}
	return u.getByID(ctx, userID)
}

func TestCreateShop(t *testing.T) {
	fixedTime := time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			if shop.ShopID == "" {
				t.Fatal("expected generated shop id")
			}
			if shop.OwnerUserID != "user-1" {
				t.Fatalf("unexpected owner: %s", shop.OwnerUserID)
			}
			if shop.Status != ShopStatusActive {
				t.Fatalf("unexpected status: %s", shop.Status)
			}
			if !shop.CreatedAt.Equal(fixedTime) || !shop.UpdatedAt.Equal(fixedTime) {
				t.Fatal("expected fixed timestamps")
			}
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	}, pledgeRepositoryStub{}, userRepositoryStub{})
	service.now = func() time.Time { return fixedTime }

	shop, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: "user-1",
		Name:        "Green Shop",
		Description: "Fresh daily",
		Address:     "123 Main St",
		Latitude:    10.762622,
		Longitude:   106.660172,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shop.ShopID == "" {
		t.Fatal("expected shop id")
	}
}

func TestUpdateRejectsNonOwner(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "owner-1"}, nil
		},
	}, pledgeRepositoryStub{}, userRepositoryStub{})

	_, err := service.Update(context.Background(), UpdateInput{
		ShopID:      "shop-1",
		OwnerUserID: "user-2",
		Name:        "Name",
		Address:     "Address",
		Latitude:    10,
		Longitude:   106,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateRejectsInvalidCoordinates(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	}, pledgeRepositoryStub{}, userRepositoryStub{})

	_, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: "user-1",
		Name:        "Shop",
		Address:     "Address",
		Latitude:    200,
		Longitude:   106,
	})
	if !errors.Is(err, ErrInvalidShop) {
		t.Fatalf("expected ErrInvalidShop, got %v", err)
	}
}

func TestListReturnsTrustSummary(t *testing.T) {
	committedAt := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	service := NewService(shopRepositoryStub{
		list: func(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
			if filter.Status != ShopStatusActive {
				t.Fatalf("unexpected status filter: %s", filter.Status)
			}
			return []domain.Shop{{ShopID: "shop-1", Name: "Green Shop", Address: "123 Main St", Status: ShopStatusActive}}, nil
		},
		save:    func(ctx context.Context, shop domain.Shop) error { return nil },
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) { return domain.Shop{}, nil },
	}, pledgeRepositoryStub{
		listByShopID: func(ctx context.Context, shopID string) ([]domain.Pledge, error) {
			return []domain.Pledge{{
				PledgeID:   "pledge-1",
				ShopID:     shopID,
				Status:     "committed",
				Score:      8.8,
				Category:   "fresh_produce",
				Confidence: 0.92,
				CreatedAt:  committedAt,
			}}, nil
		},
	}, userRepositoryStub{})

	result, err := service.List(context.Background(), ListInput{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if !result.Items[0].TrustSummary.HasPledges {
		t.Fatal("expected trust summary with pledge")
	}
	if result.Items[0].TrustSummary.LatestPledgeID != "pledge-1" {
		t.Fatalf("unexpected latest pledge id: %s", result.Items[0].TrustSummary.LatestPledgeID)
	}
}

func TestModerateRequiresAdmin(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error { return nil },
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: ShopStatusActive}, nil
		},
	}, pledgeRepositoryStub{}, userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "user"}, nil
		},
	})

	_, err := service.Moderate(context.Background(), ModerateInput{
		ShopID:          "shop-1",
		ModeratorUserID: "user-1",
		Status:          ShopStatusSuspended,
	})
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
}
