package integrity

import (
	"context"
	"testing"
	"time"

	"vngrocery/internal/domain"
)

type engagementRepoStub struct {
	counts map[string]domain.EngagementCount
}

func (r *engagementRepoStub) Save(context.Context, domain.Engagement) error { return nil }
func (r *engagementRepoStub) Delete(context.Context, string) error          { return nil }
func (r *engagementRepoStub) Has(context.Context, string) (bool, error)     { return false, nil }
func (r *engagementRepoStub) CountKind(context.Context, string, string, string) (int, error) {
	return 0, nil
}

func (r *engagementRepoStub) ListKindsByUser(context.Context, string, string, string) ([]string, error) {
	return nil, nil
}

func (r *engagementRepoStub) SaveCount(_ context.Context, count domain.EngagementCount) error {
	r.counts[count.CountID] = count
	return nil
}

func (r *engagementRepoStub) GetCount(_ context.Context, countID string) (domain.EngagementCount, error) {
	return r.counts[countID], nil
}

func (r *engagementRepoStub) ListCountsByChainAnchorStatus(_ context.Context, status string, _ int) ([]domain.EngagementCount, error) {
	var items []domain.EngagementCount
	for _, count := range r.counts {
		if count.ChainAnchorStatus == status {
			items = append(items, count)
		}
	}
	return items, nil
}

func sampleCount() domain.EngagementCount {
	return domain.EngagementCount{
		CountID:    "product:product-1",
		TargetType: domain.EngagementTargetProduct,
		TargetID:   "product-1",
		Likes:      4,
		Loves:      2,
		Version:    3,
		UpdatedAt:  time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
	}
}

func TestEngagementHashMovesWithTheFigure(t *testing.T) {
	before, err := HashEngagementCount(sampleCount())
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	// One more heart has to produce a different hash, or the anchor would keep
	// vouching for a total that has since changed.
	moved := sampleCount()
	moved.Loves++
	after, err := HashEngagementCount(moved)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if before == after {
		t.Fatal("expected the hash to change when a heart was added")
	}

	again, err := HashEngagementCount(sampleCount())
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if again != before {
		t.Fatal("expected the same figure to hash the same way twice")
	}
}

func TestPreparingACountLeavesItOwingAnAnchor(t *testing.T) {
	service := NewService(nil, nil, nil)
	count, err := service.PrepareEngagementCount(sampleCount())
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if count.DataHash == "" {
		t.Fatal("expected a data hash")
	}
	if count.ChainAnchorStatus != ChainAnchorStatusPending || count.IntegrityStatus != IntegrityStatusPendingAnchor {
		t.Fatalf("expected pending statuses, got %q and %q", count.ChainAnchorStatus, count.IntegrityStatus)
	}
	if count.ChainTxHash != "" || count.ChainAnchorTime != nil {
		t.Fatal("expected any previous anchor to be cleared")
	}
}

func TestPendingCountsAreAnchoredUnderTheirOwnRecordID(t *testing.T) {
	prepared, err := (&Service{now: time.Now}).PrepareEngagementCount(sampleCount())
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	repo := &engagementRepoStub{counts: map[string]domain.EngagementCount{prepared.CountID: prepared}}

	var sawRecordID, sawHash string
	var sawVersion int
	chain := chainStub{
		commit: func(_ context.Context, recordID, dataHash string, _ time.Time, version int) (CommitResult, error) {
			sawRecordID, sawHash, sawVersion = recordID, dataHash, version
			return CommitResult{TxHash: "0xabc", BlockNumber: 42, Mined: true}, nil
		},
		verify: func(context.Context, string, string) (bool, error) { return true, nil },
	}

	service := NewService(nil, chain, nil)
	service.SetEngagementRepository(repo)

	if err := service.ProcessPendingEngagementCounts(context.Background(), 10); err != nil {
		t.Fatalf("process: %v", err)
	}

	if sawRecordID != "engagement:product:product-1" {
		t.Fatalf("wrong record id on chain: %q", sawRecordID)
	}
	if sawHash != prepared.DataHash || sawVersion != prepared.Version {
		t.Fatalf("committed the wrong hash or version: %q %d", sawHash, sawVersion)
	}

	stored := repo.counts[prepared.CountID]
	if stored.ChainAnchorStatus != ChainAnchorStatusAnchored || stored.ChainTxHash != "0xabc" || stored.ChainBlockNumber != 42 {
		t.Fatalf("expected the count marked anchored, got %+v", stored)
	}
	if stored.ChainAnchorTime == nil {
		t.Fatal("expected an anchor time")
	}
}

func TestAFailedAnchorIsKeptForRetryRatherThanLost(t *testing.T) {
	prepared, err := (&Service{now: time.Now}).PrepareEngagementCount(sampleCount())
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	repo := &engagementRepoStub{counts: map[string]domain.EngagementCount{prepared.CountID: prepared}}

	chain := chainStub{
		commit: func(context.Context, string, string, time.Time, int) (CommitResult, error) {
			return CommitResult{}, context.DeadlineExceeded
		},
		verify: func(context.Context, string, string) (bool, error) { return false, nil },
	}
	service := NewService(nil, chain, nil)
	service.SetEngagementRepository(repo)

	if err := service.ProcessPendingEngagementCounts(context.Background(), 10); err != nil {
		t.Fatalf("process should absorb a chain failure, got %v", err)
	}

	stored := repo.counts[prepared.CountID]
	if stored.ChainAnchorStatus != ChainAnchorStatusPending {
		t.Fatalf("expected the count still pending, got %q", stored.ChainAnchorStatus)
	}
	if stored.ChainAnchorAttempts != 1 || stored.ChainAnchorNextAttemptAt == nil {
		t.Fatalf("expected a retry booked, got %+v", stored)
	}
	if stored.ChainAnchorLastError == "" {
		t.Fatal("expected the failure recorded")
	}
}
