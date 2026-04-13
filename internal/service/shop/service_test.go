package shop

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
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
	getByID      func(ctx context.Context, pledgeID string) (domain.Pledge, error)
}

func (p pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error { return nil }
func (p pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	if p.getByID == nil {
		return domain.Pledge{}, nil
	}
	return p.getByID(ctx, pledgeID)
}
func (p pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	if p.listByShopID == nil {
		return nil, nil
	}
	return p.listByShopID(ctx, shopID)
}
func (p pledgeRepositoryStub) ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
	return nil, nil
}

type buyerCheckRepositoryStub struct {
	save              func(ctx context.Context, check domain.BuyerCheck) error
	getByID           func(ctx context.Context, checkID string) (domain.BuyerCheck, error)
	listByShopID      func(ctx context.Context, shopID string) ([]domain.BuyerCheck, error)
	listByBuyerUserID func(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error)
	list              func(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error)
}

func (b buyerCheckRepositoryStub) Save(ctx context.Context, check domain.BuyerCheck) error {
	if b.save == nil {
		return nil
	}
	return b.save(ctx, check)
}

func (b buyerCheckRepositoryStub) GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error) {
	if b.getByID != nil {
		return b.getByID(ctx, checkID)
	}
	return domain.BuyerCheck{}, nil
}

func (b buyerCheckRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
	if b.listByShopID == nil {
		return nil, nil
	}
	return b.listByShopID(ctx, shopID)
}

func (b buyerCheckRepositoryStub) ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
	if b.listByBuyerUserID != nil {
		return b.listByBuyerUserID(ctx, buyerUserID)
	}
	return nil, nil
}

func (b buyerCheckRepositoryStub) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	if b.list != nil {
		return b.list(ctx, filter)
	}
	return nil, nil
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
func (u userRepositoryStub) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	return nil, errors.New("not implemented")
}

type integrityReaderStub struct {
	get func(ctx context.Context, pledge domain.Pledge) (PledgeIntegrityView, error)
}

func (s integrityReaderStub) GetPledgeIntegrity(ctx context.Context, pledge domain.Pledge) (PledgeIntegrityView, error) {
	return s.get(ctx, pledge)
}

func (s integrityReaderStub) VerifyPledgeHash(ctx context.Context, pledge domain.Pledge, dataHash string) (PledgeIntegrityView, error) {
	return s.get(ctx, pledge)
}

func (s integrityReaderStub) ReanchorPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	return pledge, nil
}

func (s integrityReaderStub) RevokePledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	return pledge, nil
}

type reviewRepositoryStub struct {
	save             func(ctx context.Context, review domain.ShopReview) error
	getByShopAndUser func(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error)
	listByShopID     func(ctx context.Context, shopID string) ([]domain.ShopReview, error)
}

func (r reviewRepositoryStub) Save(ctx context.Context, review domain.ShopReview) error {
	if r.save == nil {
		return nil
	}
	return r.save(ctx, review)
}

func (r reviewRepositoryStub) GetByShopAndUser(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error) {
	if r.getByShopAndUser == nil {
		return domain.ShopReview{}, nil
	}
	return r.getByShopAndUser(ctx, shopID, reviewerUserID)
}

func (r reviewRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	if r.listByShopID == nil {
		return nil, nil
	}
	return r.listByShopID(ctx, shopID)
}

type auditLoggerStub struct {
	log     func(ctx context.Context, input audit.Input) error
	logHits int
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	if s.log != nil {
		return s.log(ctx, input)
	}
	return nil
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
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)
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
	if shop.Version != 1 {
		t.Fatalf("expected version 1, got %d", shop.Version)
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
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)

	_, err := service.Update(context.Background(), UpdateInput{
		ShopID:          "shop-1",
		OwnerUserID:     "user-2",
		ExpectedVersion: 1,
		Name:            "Name",
		Address:         "Address",
		Latitude:        10,
		Longitude:       106,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateRejectsVersionConflict(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			t.Fatal("save should not be called")
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1", Status: ShopStatusActive, Version: 4}, nil
		},
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)

	_, err := service.Update(context.Background(), UpdateInput{
		ShopID:          "shop-1",
		OwnerUserID:     "user-1",
		ExpectedVersion: 3,
		Name:            "Name",
		Address:         "Address",
		Latitude:        10,
		Longitude:       106,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestDeleteMarksShopDeleted(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var savedShop domain.Shop
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error {
			savedShop = shop
			return nil
		},
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1", Status: ShopStatusActive, Version: 1}, nil
		},
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, auditLogger)

	shop, err := service.Delete(context.Background(), DeleteInput{ShopID: "shop-1", OwnerUserID: "user-1", ExpectedVersion: 1})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shop.Status != ShopStatusDeleted || savedShop.Status != ShopStatusDeleted {
		t.Fatalf("expected deleted status, got %#v %#v", shop, savedShop)
	}
	if shop.Version != 2 || savedShop.Version != 2 {
		t.Fatalf("expected deleted version 2, got shop=%d saved=%d", shop.Version, savedShop.Version)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
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
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)

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
	}, buyerCheckRepositoryStub{
		listByShopID: func(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
			return []domain.BuyerCheck{
				{ShopID: shopID, Verdict: "trusted", Trusted: true, CategoryMatch: true, ScoreDeltaAbs: 0.4},
				{ShopID: shopID, Verdict: "warning", CategoryMatch: true, ScoreDeltaAbs: 1.8},
			}, nil
		},
	}, reviewRepositoryStub{
		listByShopID: func(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
			return []domain.ShopReview{
				{ShopID: shopID, Rating: 5},
				{ShopID: shopID, Rating: 3},
			}, nil
		},
	}, userRepositoryStub{}, nil)

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
	if result.Items[0].RatingSummary.RatingCount != 2 {
		t.Fatalf("expected rating count 2, got %d", result.Items[0].RatingSummary.RatingCount)
	}
	if result.Items[0].RatingSummary.AverageRating != 4 {
		t.Fatalf("expected average rating 4, got %v", result.Items[0].RatingSummary.AverageRating)
	}
	if result.Items[0].TrustSummary.Score <= 0 {
		t.Fatalf("expected trust score, got %v", result.Items[0].TrustSummary.Score)
	}
	if result.Items[0].TrustSummary.BuyerCheckCount != 2 {
		t.Fatalf("expected buyer check count 2, got %d", result.Items[0].TrustSummary.BuyerCheckCount)
	}
	if result.Items[0].TrustSummary.TrustedCheckCount != 1 {
		t.Fatalf("expected trusted check count 1, got %d", result.Items[0].TrustSummary.TrustedCheckCount)
	}
	if result.Items[0].TrustSummary.FormulaVersion != "trust_score_v2" {
		t.Fatalf("unexpected formula version: %s", result.Items[0].TrustSummary.FormulaVersion)
	}
	if result.Items[0].TrustSummary.CoverageScore <= 0 || result.Items[0].TrustSummary.RecencyScore <= 0 {
		t.Fatalf("expected v2 breakdown scores, got %#v", result.Items[0].TrustSummary)
	}
}

func TestListPledgesFiltersByProductAndCategory(t *testing.T) {
	service := NewService(shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: ShopStatusActive}, nil
		},
	}, pledgeRepositoryStub{
		listByShopID: func(ctx context.Context, shopID string) ([]domain.Pledge, error) {
			return []domain.Pledge{
				{PledgeID: "pledge-1", ShopID: shopID, ProductID: "product-1", Category: "fresh_produce", Score: 8.5},
				{PledgeID: "pledge-2", ShopID: shopID, ProductID: "product-2", Category: "fresh_produce", Score: 7.2},
				{PledgeID: "pledge-3", ShopID: shopID, ProductID: "product-1", Category: "stale_fish", Score: 4.1},
			}, nil
		},
	}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)

	pledges, err := service.ListPledges(context.Background(), PledgeHistoryInput{
		ShopID:    "shop-1",
		ProductID: "product-1",
		Category:  "FRESH_PRODUCE",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pledges) != 1 || pledges[0].PledgeID != "pledge-1" {
		t.Fatalf("unexpected pledges: %#v", pledges)
	}
}

func TestGetPledgeIntegrityUsesReader(t *testing.T) {
	service := NewService(shopRepositoryStub{}, pledgeRepositoryStub{
		getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
			return domain.Pledge{
				PledgeID:          pledgeID,
				ShopID:            "shop-1",
				DataHash:          "data-hash",
				ChainAnchorStatus: "anchored",
				IntegrityStatus:   "anchored",
			}, nil
		},
	}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)
	service.SetPledgeIntegrityReader(integrityReaderStub{
		get: func(ctx context.Context, pledge domain.Pledge) (PledgeIntegrityView, error) {
			return PledgeIntegrityView{
				PledgeID:          pledge.PledgeID,
				ShopID:            pledge.ShopID,
				DataHash:          pledge.DataHash,
				ChainAnchorStatus: pledge.ChainAnchorStatus,
				IntegrityStatus:   pledge.IntegrityStatus,
				OnChainMatch:      true,
			}, nil
		},
	})

	result, err := service.GetPledgeIntegrity(context.Background(), PledgeIntegrityInput{
		ShopID:   "shop-1",
		PledgeID: "pledge-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.PledgeID != "pledge-1" || !result.OnChainMatch || result.DataHash != "data-hash" {
		t.Fatalf("unexpected integrity result: %#v", result)
	}
}

func TestGetPledgeProofBuildsViewerBundle(t *testing.T) {
	service := NewService(shopRepositoryStub{}, pledgeRepositoryStub{
		getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
			return domain.Pledge{
				PledgeID:          pledgeID,
				ShopID:            "shop-1",
				ProductID:         "product-1",
				Score:             8.9,
				Category:          "fresh_produce",
				Confidence:        0.93,
				ImageHash:         "hash-1",
				ImageCID:          "cid-1",
				DataHash:          "data-hash",
				ChainAnchorStatus: "anchored",
				IntegrityStatus:   "anchored",
			}, nil
		},
	}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, nil)
	service.SetPledgeIntegrityReader(integrityReaderStub{
		get: func(ctx context.Context, pledge domain.Pledge) (PledgeIntegrityView, error) {
			return PledgeIntegrityView{
				PledgeID:          pledge.PledgeID,
				ShopID:            pledge.ShopID,
				DataHash:          pledge.DataHash,
				ChainAnchorStatus: pledge.ChainAnchorStatus,
				IntegrityStatus:   pledge.IntegrityStatus,
				OnChainMatch:      true,
			}, nil
		},
	})

	result, err := service.GetPledgeProof(context.Background(), PledgeIntegrityInput{
		ShopID:   "shop-1",
		PledgeID: "pledge-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ProofStatus != "verified" {
		t.Fatalf("unexpected proof status: %#v", result)
	}
	if len(result.RecommendedActions) == 0 || result.Integrity.PledgeID != "pledge-1" {
		t.Fatalf("unexpected proof bundle: %#v", result)
	}
}

func TestModerateRequiresAdmin(t *testing.T) {
	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error { return nil },
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: ShopStatusActive}, nil
		},
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: "user"}, nil
		},
	}, nil)

	_, err := service.Moderate(context.Background(), ModerateInput{
		ShopID:          "shop-1",
		ModeratorUserID: "user-1",
		ExpectedVersion: 1,
		Status:          ShopStatusSuspended,
	})
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
}

func TestReviewCreatesOrUpdatesRating(t *testing.T) {
	fixedTime := time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)
	service := NewService(shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, Status: ShopStatusActive}, nil
		},
		save: func(ctx context.Context, shop domain.Shop) error { return nil },
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{
		save: func(ctx context.Context, review domain.ShopReview) error {
			if review.Rating != 5 {
				t.Fatalf("unexpected rating: %d", review.Rating)
			}
			return nil
		},
		getByShopAndUser: func(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error) {
			return domain.ShopReview{}, nil
		},
	}, userRepositoryStub{}, nil)
	service.now = func() time.Time { return fixedTime }

	review, err := service.Review(context.Background(), ReviewInput{
		ShopID:         "shop-1",
		ReviewerUserID: "user-1",
		Rating:         5,
		Comment:        "Very good",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if review.ReviewID == "" {
		t.Fatal("expected generated review id")
	}
	if review.Status != ReviewStatusActive {
		t.Fatalf("unexpected review status: %s", review.Status)
	}
	if review.Version != 1 {
		t.Fatalf("unexpected review version: %d", review.Version)
	}
}

func TestCreateShopWritesAuditLog(t *testing.T) {
	auditLogger := &auditLoggerStub{
		log: func(ctx context.Context, input audit.Input) error {
			if input.Action != "shop.created" || input.Status != "created" || input.ResourceVersion != 1 || input.ActorUserID != "user-1" || input.ResourceType != "shop" {
				t.Fatalf("unexpected audit input: %#v", input)
			}
			return nil
		},
	}

	service := NewService(shopRepositoryStub{
		save: func(ctx context.Context, shop domain.Shop) error { return nil },
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	}, pledgeRepositoryStub{}, buyerCheckRepositoryStub{}, reviewRepositoryStub{}, userRepositoryStub{}, auditLogger)

	if _, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: "user-1",
		Name:        "Green Shop",
		Address:     "123 Main St",
		Latitude:    10,
		Longitude:   106,
	}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}
