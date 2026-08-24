package engagement

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/service/audit"
)

type repoStub struct {
	marks  map[string]domain.Engagement
	counts map[string]domain.EngagementCount
}

func newRepoStub() *repoStub {
	return &repoStub{marks: map[string]domain.Engagement{}, counts: map[string]domain.EngagementCount{}}
}

func (r *repoStub) Save(_ context.Context, mark domain.Engagement) error {
	r.marks[mark.EngagementID] = mark
	return nil
}

func (r *repoStub) Delete(_ context.Context, id string) error {
	delete(r.marks, id)
	return nil
}

func (r *repoStub) Has(_ context.Context, id string) (bool, error) {
	_, ok := r.marks[id]
	return ok, nil
}

func (r *repoStub) CountKind(_ context.Context, targetType, targetID, kind string) (int, error) {
	total := 0
	for _, mark := range r.marks {
		if mark.TargetType == targetType && mark.TargetID == targetID && mark.Kind == kind {
			total++
		}
	}
	return total, nil
}

func (r *repoStub) ListKindsByUser(_ context.Context, userID, targetType, targetID string) ([]string, error) {
	kinds := []string{}
	for _, mark := range r.marks {
		if mark.UserID == userID && mark.TargetType == targetType && mark.TargetID == targetID {
			kinds = append(kinds, mark.Kind)
		}
	}
	return kinds, nil
}

func (r *repoStub) SaveCount(_ context.Context, count domain.EngagementCount) error {
	r.counts[count.CountID] = count
	return nil
}

func (r *repoStub) GetCount(_ context.Context, countID string) (domain.EngagementCount, error) {
	return r.counts[countID], nil
}

func (r *repoStub) ListCountsByChainAnchorStatus(_ context.Context, status string, _ int) ([]domain.EngagementCount, error) {
	var items []domain.EngagementCount
	for _, count := range r.counts {
		if count.ChainAnchorStatus == status {
			items = append(items, count)
		}
	}
	return items, nil
}

type anchorStub struct{ prepared int }

func (a *anchorStub) PrepareEngagementCount(count domain.EngagementCount) (domain.EngagementCount, error) {
	a.prepared++
	count.DataHash = "hash"
	count.ChainAnchorStatus = "pending"
	return count, nil
}

type auditStub struct{ inputs []audit.Input }

func (a *auditStub) Log(_ context.Context, input audit.Input) error {
	a.inputs = append(a.inputs, input)
	return nil
}

func newService(anchorer Anchorer, logger AuditLogger) (*Service, *repoStub) {
	repo := newRepoStub()
	service := NewService(repo, nil, nil, logger, anchorer)
	service.now = func() time.Time { return time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC) }
	return service, repo
}

func TestTogglingTwiceTakesTheMarkBack(t *testing.T) {
	service, _ := newService(nil, nil)
	ctx := context.Background()

	state, err := service.Toggle(ctx, "user-1", domain.EngagementTargetShop, "shop-1", domain.EngagementFollow)
	if err != nil {
		t.Fatalf("follow: %v", err)
	}
	if state.Follows != 1 || len(state.Mine) != 1 {
		t.Fatalf("expected one follow held by the caller, got %d and %v", state.Follows, state.Mine)
	}

	state, err = service.Toggle(ctx, "user-1", domain.EngagementTargetShop, "shop-1", domain.EngagementFollow)
	if err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if state.Follows != 0 || len(state.Mine) != 0 {
		t.Fatalf("expected the follow taken back, got %d and %v", state.Follows, state.Mine)
	}
}

func TestTotalsAreCountedNotIncremented(t *testing.T) {
	service, repo := newService(nil, nil)
	ctx := context.Background()

	for _, user := range []string{"user-1", "user-2", "user-3"} {
		if _, err := service.Toggle(ctx, user, domain.EngagementTargetProduct, "product-1", domain.EngagementLove); err != nil {
			t.Fatalf("love from %s: %v", user, err)
		}
	}

	// A row removed behind the service's back is still not counted: the total
	// is a count of the marks, never a running tally that could drift.
	delete(repo.marks, markID("user-2", domain.EngagementTargetProduct, "product-1", domain.EngagementLove))

	state, err := service.Toggle(ctx, "user-4", domain.EngagementTargetProduct, "product-1", domain.EngagementLike)
	if err != nil {
		t.Fatalf("like: %v", err)
	}
	if state.Loves != 2 {
		t.Fatalf("expected 2 loves after one row vanished, got %d", state.Loves)
	}
	if state.Likes != 1 {
		t.Fatalf("expected 1 like, got %d", state.Likes)
	}
}

func TestEveryChangeQueuesAnAnchorAndASignedEvent(t *testing.T) {
	anchorer := &anchorStub{}
	logger := &auditStub{}
	service, repo := newService(anchorer, logger)
	ctx := context.Background()

	if _, err := service.Toggle(ctx, "user-1", domain.EngagementTargetShop, "shop-1", domain.EngagementFollow); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if _, err := service.Toggle(ctx, "user-1", domain.EngagementTargetShop, "shop-1", domain.EngagementFollow); err != nil {
		t.Fatalf("unfollow: %v", err)
	}

	if anchorer.prepared != 2 {
		t.Fatalf("expected both changes queued for the chain, got %d", anchorer.prepared)
	}
	count := repo.counts["shop:shop-1"]
	if count.ChainAnchorStatus != "pending" || count.DataHash == "" {
		t.Fatalf("expected the total to owe an anchor, got %+v", count)
	}
	// The version has to move with the figure, or the chain would be asked to
	// commit a new hash under a version it has already recorded.
	if count.Version != 2 {
		t.Fatalf("expected version 2 after two changes, got %d", count.Version)
	}

	if len(logger.inputs) != 2 {
		t.Fatalf("expected a signed event per tap, got %d", len(logger.inputs))
	}
	if logger.inputs[0].Action != "engagement.added" || logger.inputs[1].Action != "engagement.removed" {
		t.Fatalf("expected added then removed, got %s then %s", logger.inputs[0].Action, logger.inputs[1].Action)
	}
	if logger.inputs[0].ResourceType != "engagement" || logger.inputs[0].ResourceID != "shop:shop-1" {
		t.Fatalf("event points at the wrong resource: %+v", logger.inputs[0])
	}
}

func TestAShopCannotBeLovedAndAProductCannotBeFollowed(t *testing.T) {
	service, _ := newService(nil, nil)
	ctx := context.Background()

	if _, err := service.Toggle(ctx, "user-1", domain.EngagementTargetShop, "shop-1", domain.EngagementLove); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind for a loved shop, got %v", err)
	}
	if _, err := service.Toggle(ctx, "user-1", domain.EngagementTargetProduct, "product-1", domain.EngagementFollow); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind for a followed product, got %v", err)
	}
	if _, err := service.Toggle(ctx, "user-1", "basket", "b-1", domain.EngagementLike); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected ErrInvalidTarget, got %v", err)
	}
}

func TestUnmarkedTargetReadsAsZeroRatherThanFailing(t *testing.T) {
	service, _ := newService(nil, nil)

	state, err := service.Get(context.Background(), "user-1", domain.EngagementTargetProduct, "product-9")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state.Likes != 0 || state.Loves != 0 || len(state.Mine) != 0 {
		t.Fatalf("expected an untouched product to read as zero, got %+v", state)
	}
	if state.TargetID != "product-9" {
		t.Fatalf("expected the target echoed back, got %q", state.TargetID)
	}
}
