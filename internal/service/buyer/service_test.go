package buyer

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
	visionservice "vngrocery/internal/service/vision"
)

type pledgeRepositoryStub struct {
	getByID func(ctx context.Context, pledgeID string) (domain.Pledge, error)
}

func (s pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error {
	return nil
}

func (s pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	if s.getByID == nil {
		return domain.Pledge{}, errors.New("not implemented")
	}
	return s.getByID(ctx, pledgeID)
}

func (s pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	return nil, nil
}

func (s pledgeRepositoryStub) ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
	return nil, nil
}

type buyerCheckRepositoryStub struct {
	save              func(ctx context.Context, check domain.BuyerCheck) error
	getByID           func(ctx context.Context, checkID string) (domain.BuyerCheck, error)
	listByShopID      func(ctx context.Context, shopID string) ([]domain.BuyerCheck, error)
	listByBuyerUserID func(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error)
	list              func(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error)
}

func (s buyerCheckRepositoryStub) Save(ctx context.Context, check domain.BuyerCheck) error {
	if s.save == nil {
		return nil
	}
	return s.save(ctx, check)
}

func (s buyerCheckRepositoryStub) GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error) {
	if s.getByID != nil {
		return s.getByID(ctx, checkID)
	}
	return domain.BuyerCheck{}, errors.New("not implemented")
}

func (s buyerCheckRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
	if s.listByShopID != nil {
		return s.listByShopID(ctx, shopID)
	}
	return nil, nil
}

func (s buyerCheckRepositoryStub) ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
	if s.listByBuyerUserID != nil {
		return s.listByBuyerUserID(ctx, buyerUserID)
	}
	return nil, nil
}

func (s buyerCheckRepositoryStub) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, nil
}

type userRepositoryStub struct{}

func (userRepositoryStub) Save(ctx context.Context, user domain.User) error { return nil }
func (userRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	return domain.User{UserID: userID, Role: "admin"}, nil
}
func (userRepositoryStub) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	return nil, nil
}

type scorerStub struct {
	score func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error)
}

func (s scorerStub) Score(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
	return s.score(ctx, input)
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

func TestCheckReturnsTrustedVerdict(t *testing.T) {
	auditLogger := &auditLoggerStub{
		log: func(ctx context.Context, input audit.Input) error {
			if input.ActorUserID != "buyer-1" || input.ResourceType != "buyer_check" || input.Action != "buyer_check.completed" || input.Status != "completed" {
				t.Fatalf("unexpected audit input: %#v", input)
			}
			return nil
		},
	}

	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				return domain.Pledge{
					PledgeID:   pledgeID,
					ShopID:     "shop-1",
					Score:      8.5,
					Category:   "fresh_produce",
					Confidence: 0.91,
				}, nil
			},
		},
		buyerCheckRepositoryStub{
			save: func(ctx context.Context, check domain.BuyerCheck) error {
				if check.ShopID != "shop-1" || check.PledgeID != "pledge-1" || check.BuyerUserID != "buyer-1" || check.ImageHash != "image-hash-1" || !check.Trusted {
					t.Fatalf("unexpected persisted buyer check: %#v", check)
				}
				return nil
			},
		},
		userRepositoryStub{},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      8.1,
					Category:   "fresh_produce",
					Confidence: 0.88,
				}, nil
			},
		},
		auditLogger,
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID:    "pledge-1",
		BuyerUserID: "buyer-1",
		ImageHash:   "image-hash-1",
		Image: visionservice.ImageInput{
			Filename: "shop.jpg",
			Content:  bytes.NewBuffer([]byte("fake")),
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !result.Trusted {
		t.Fatal("expected trusted result")
	}
	if !result.HasPledge {
		t.Fatal("expected result to include seller pledge")
	}
	if result.Verdict != "trusted" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
	if result.PolicyVersion != policyVersionV1 {
		t.Fatalf("unexpected policy version: %s", result.PolicyVersion)
	}
	if !result.CategoryMatch {
		t.Fatal("expected category match")
	}
	if result.ScoreDeltaAbs <= 0 {
		t.Fatal("expected positive score delta abs")
	}
	if len(result.Reasons) != 0 {
		t.Fatalf("expected no warning reasons for trusted result, got %v", result.Reasons)
	}
	if result.CheckID == "" {
		t.Fatal("expected check id")
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestCheckReturnsHighRiskOnMismatch(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				return domain.Pledge{
					PledgeID: pledgeID,
					ShopID:   "shop-1",
					Score:    9.0,
					Category: "fresh_produce",
				}, nil
			},
		},
		buyerCheckRepositoryStub{},
		userRepositoryStub{},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      4.2,
					Category:   "poor_storage",
					Confidence: 0.9,
				}, nil
			},
		},
		nil,
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID:    "pledge-1",
		BuyerUserID: "buyer-1",
		Image: visionservice.ImageInput{
			Filename: "shop.jpg",
			Content:  bytes.NewBuffer([]byte("fake")),
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Verdict != "high_risk" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
	if result.CategoryMatch {
		t.Fatal("expected category mismatch")
	}
	if len(result.Reasons) == 0 {
		t.Fatal("expected risk reasons")
	}
}

func TestCheckReturnsWarningForLowConfidence(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				return domain.Pledge{
					PledgeID: pledgeID,
					ShopID:   "shop-1",
					Score:    7.5,
					Category: "fresh_produce",
				}, nil
			},
		},
		buyerCheckRepositoryStub{},
		userRepositoryStub{},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      7.1,
					Category:   "fresh_produce",
					Confidence: 0.42,
				}, nil
			},
		},
		nil,
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID:    "pledge-1",
		BuyerUserID: "buyer-1",
		Image: visionservice.ImageInput{
			Filename: "shop.jpg",
			Content:  bytes.NewBuffer([]byte("fake")),
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Verdict != "warning" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
	if result.Trusted {
		t.Fatal("expected untrusted warning result")
	}
	if len(result.Reasons) == 0 {
		t.Fatal("expected warning reasons")
	}
	found := false
	for _, reason := range result.Reasons {
		if reason == "low_ai_confidence" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected low_ai_confidence in reasons, got %v", result.Reasons)
	}
}

func TestCheckWithoutPledgeReturnsStandaloneQuality(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				t.Fatal("repository should not be called")
				return domain.Pledge{}, nil
			},
		},
		buyerCheckRepositoryStub{},
		userRepositoryStub{},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      6.2,
					Category:   "bruised_fruit",
					Confidence: 0.86,
				}, nil
			},
		},
		nil,
	)

	result, err := service.Check(context.Background(), CheckInput{
		BuyerUserID: "buyer-1",
		ImageHash:   "image-hash-1",
		Image: visionservice.ImageInput{
			Filename: "buyer.jpg",
			Content:  bytes.NewBuffer([]byte("fake")),
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.HasPledge {
		t.Fatal("expected result without seller pledge")
	}
	if result.Trusted {
		t.Fatal("standalone quality check should not be marked trusted")
	}
	if result.Verdict != "no_pledge" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
	if result.ActualScore != 6.2 || result.ActualCategory != "bruised_fruit" {
		t.Fatalf("unexpected actual quality result: %#v", result)
	}
	if len(result.Reasons) != 1 || result.Reasons[0] != "no_seller_pledge" {
		t.Fatalf("unexpected reasons: %v", result.Reasons)
	}
	if result.CheckID == "" || result.BuyerUserID != "buyer-1" || result.ImageHash != "image-hash-1" {
		t.Fatalf("expected persisted standalone check metadata, got %#v", result)
	}
}

func TestCheckRejectsWhenQuotaExceeded(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{},
		buyerCheckRepositoryStub{
			listByBuyerUserID: func(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
				now := time.Now().UTC()
				items := make([]domain.BuyerCheck, 0, 10)
				for i := 0; i < 10; i++ {
					items = append(items, domain.BuyerCheck{CheckID: "check", CreatedAt: now})
				}
				return items, nil
			},
		},
		userRepositoryStub{},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{}, nil
			},
		},
		nil,
	)

	_, err := service.Check(context.Background(), CheckInput{
		BuyerUserID: "buyer-1",
		Image: visionservice.ImageInput{
			Filename: "buyer.jpg",
			Content:  bytes.NewBuffer([]byte("fake")),
		},
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestListBuyerChecksForAdmin(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{},
		buyerCheckRepositoryStub{
			list: func(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
				if filter.ShopID != "shop-1" || filter.Status != "completed" {
					t.Fatalf("unexpected filter: %#v", filter)
				}
				return []domain.BuyerCheck{
					{CheckID: "check-1", Status: "completed"},
					{CheckID: "check-2", Status: "completed"},
				}, nil
			},
		},
		userRepositoryStub{},
		nil,
		nil,
	)

	result, err := service.List(context.Background(), ListInput{
		ActorUserID: "admin-1",
		ShopID:      "shop-1",
		Status:      "completed",
		Page:        1,
		PageSize:    1,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].CheckID != "check-1" {
		t.Fatalf("unexpected list result: %#v", result)
	}
}
