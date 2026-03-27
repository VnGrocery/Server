package buyer

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
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
	save func(ctx context.Context, check domain.BuyerCheck) error
}

func (s buyerCheckRepositoryStub) Save(ctx context.Context, check domain.BuyerCheck) error {
	if s.save == nil {
		return nil
	}
	return s.save(ctx, check)
}

func (s buyerCheckRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
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
