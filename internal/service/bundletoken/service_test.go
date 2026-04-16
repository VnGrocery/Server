package bundletoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
)

type replayRepoStub struct {
	reserve func(ctx context.Context, usage domain.BundleTokenUse) (bool, error)
}

func (s replayRepoStub) Reserve(ctx context.Context, usage domain.BundleTokenUse) (bool, error) {
	if s.reserve == nil {
		return true, nil
	}
	return s.reserve(ctx, usage)
}

func TestIssueAndVerifyBundleToken(t *testing.T) {
	service := NewService("secret", "vngrocery", 2*time.Minute, replayRepoStub{})
	service.now = func() time.Time {
		return time.Date(2026, 4, 16, 8, 0, 0, 0, time.UTC)
	}

	token, exp, err := service.Issue(IssueInput{
		ShopID:      "shop-1",
		ProductID:   "product-1",
		BundleID:    "bundle-1",
		PledgeID:    "pledge-1",
		CreatedByID: "seller-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if exp.IsZero() {
		t.Fatal("expected non-zero expiry")
	}

	claims, err := service.VerifyAndConsume(context.Background(), VerifyInput{
		Token:            token,
		BuyerUserID:      "buyer-1",
		ExpectedBundleID: "bundle-1",
		ExpectedPledgeID: "pledge-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if claims.ShopID != "shop-1" || claims.BundleID != "bundle-1" || claims.PledgeID != "pledge-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestVerifyRejectsReplay(t *testing.T) {
	calls := 0
	service := NewService("secret", "vngrocery", 2*time.Minute, replayRepoStub{
		reserve: func(ctx context.Context, usage domain.BundleTokenUse) (bool, error) {
			calls++
			return calls == 1, nil
		},
	})
	token, _, err := service.Issue(IssueInput{
		ShopID:    "shop-1",
		BundleID:  "bundle-1",
		PledgeID:  "pledge-1",
		ProductID: "product-1",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	_, err = service.VerifyAndConsume(context.Background(), VerifyInput{
		Token:            token,
		ExpectedBundleID: "bundle-1",
		ExpectedPledgeID: "pledge-1",
	})
	if err != nil {
		t.Fatalf("first verify should pass: %v", err)
	}
	_, err = service.VerifyAndConsume(context.Background(), VerifyInput{
		Token:            token,
		ExpectedBundleID: "bundle-1",
		ExpectedPledgeID: "pledge-1",
	})
	if !errors.Is(err, ErrReplayToken) {
		t.Fatalf("expected ErrReplayToken, got %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	now := time.Date(2026, 4, 16, 8, 0, 0, 0, time.UTC)
	service := NewService("secret", "vngrocery", 30*time.Second, replayRepoStub{})
	service.now = func() time.Time { return now }
	token, _, err := service.Issue(IssueInput{
		ShopID:   "shop-1",
		BundleID: "bundle-1",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	service.now = func() time.Time { return now.Add(31 * time.Second) }
	_, err = service.VerifyAndConsume(context.Background(), VerifyInput{
		Token:            token,
		ExpectedBundleID: "bundle-1",
	})
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
