package seller

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
)

type pledgeRepositoryStub struct {
	save func(ctx context.Context, pledge domain.Pledge) error
	get  func(ctx context.Context, pledgeID string) (domain.Pledge, error)
}

func (s pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error {
	return s.save(ctx, pledge)
}

func (s pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	if s.get == nil {
		return domain.Pledge{}, errors.New("not implemented")
	}
	return s.get(ctx, pledgeID)
}

func TestCommitCreatesPledge(t *testing.T) {
	fixedTime := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			if pledge.ShopID != "shop-1" {
				t.Fatalf("unexpected shop id: %s", pledge.ShopID)
			}
			if pledge.CreatedByUserID != "user-1" {
				t.Fatalf("unexpected user id: %s", pledge.CreatedByUserID)
			}
			if pledge.Status != PledgeStatusCommitted {
				t.Fatalf("unexpected status: %s", pledge.Status)
			}
			if pledge.Score != 8.8 {
				t.Fatalf("unexpected score: %v", pledge.Score)
			}
			if pledge.Category != "fresh_produce" {
				t.Fatalf("unexpected category: %s", pledge.Category)
			}
			if pledge.Confidence != 0.93 {
				t.Fatalf("unexpected confidence: %v", pledge.Confidence)
			}
			if pledge.PledgeID == "" {
				t.Fatal("expected generated pledge id")
			}
			if !pledge.CreatedAt.Equal(fixedTime) || !pledge.UpdatedAt.Equal(fixedTime) {
				t.Fatal("expected timestamps to use injected clock")
			}
			return nil
		},
	})
	service.now = func() time.Time { return fixedTime }

	pledge, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "shop-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.PledgeID == "" {
		t.Fatal("expected pledge id in response")
	}
}

func TestCommitRejectsInvalidInput(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			t.Fatal("save should not be called for invalid input")
			return nil
		},
	})

	_, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
	})
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}
