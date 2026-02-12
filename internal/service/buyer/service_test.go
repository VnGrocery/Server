package buyer

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
	visionservice "vngrocery/internal/service/vision"
)

type pledgeRepositoryStub struct {
	getByID func(ctx context.Context, pledgeID string) (domain.Pledge, error)
}

func (s pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error {
	return nil
}

func (s pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	return s.getByID(ctx, pledgeID)
}

func (s pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	return nil, nil
}

type scorerStub struct {
	score func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error)
}

func (s scorerStub) Score(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
	return s.score(ctx, input)
}

func TestCheckReturnsTrustedVerdict(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				return domain.Pledge{
					PledgeID:   pledgeID,
					Score:      8.5,
					Category:   "fresh_produce",
					Confidence: 0.91,
				}, nil
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
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID: "pledge-1",
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
}

func TestCheckReturnsHighRiskOnMismatch(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				return domain.Pledge{
					PledgeID: pledgeID,
					Score:    9.0,
					Category: "fresh_produce",
				}, nil
			},
		},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      4.2,
					Category:   "poor_storage",
					Confidence: 0.9,
				}, nil
			},
		},
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID: "pledge-1",
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
					Score:    7.5,
					Category: "fresh_produce",
				}, nil
			},
		},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				return visionservice.ScoreResult{
					Score:      7.1,
					Category:   "fresh_produce",
					Confidence: 0.42,
				}, nil
			},
		},
	)

	result, err := service.Check(context.Background(), CheckInput{
		PledgeID: "pledge-1",
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

func TestCheckRejectsMissingPledgeID(t *testing.T) {
	service := NewService(
		pledgeRepositoryStub{
			getByID: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
				t.Fatal("repository should not be called")
				return domain.Pledge{}, nil
			},
		},
		scorerStub{
			score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
				t.Fatal("scorer should not be called")
				return visionservice.ScoreResult{}, nil
			},
		},
	)

	_, err := service.Check(context.Background(), CheckInput{})
	if !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("expected ErrInvalidCheck, got %v", err)
	}
}
