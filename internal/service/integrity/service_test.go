package integrity

import (
	"context"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/service/audit"
)

type pledgeRepositoryStub struct {
	save                    func(ctx context.Context, pledge domain.Pledge) error
	getByID                 func(ctx context.Context, pledgeID string) (domain.Pledge, error)
	listByChainAnchorStatus func(ctx context.Context, status string, limit int) ([]domain.Pledge, error)
}

func (s pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error {
	if s.save != nil {
		return s.save(ctx, pledge)
	}
	return nil
}

func (s pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	if s.getByID != nil {
		return s.getByID(ctx, pledgeID)
	}
	return domain.Pledge{}, nil
}

func (s pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	return nil, nil
}

func (s pledgeRepositoryStub) ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
	if s.listByChainAnchorStatus != nil {
		return s.listByChainAnchorStatus(ctx, status, limit)
	}
	return nil, nil
}

type chainStub struct {
	commit  func(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (CommitResult, error)
	revoke  func(ctx context.Context, recordID string, version int) (CommitResult, error)
	verify  func(ctx context.Context, recordID, dataHash string) (bool, error)
	latest  func(ctx context.Context, recordID string) (LatestRecord, error)
	receipt func(ctx context.Context, txHash string) (CommitResult, error)
}

func (s chainStub) CommitHash(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (CommitResult, error) {
	return s.commit(ctx, recordID, dataHash, timestamp, version)
}

func (s chainStub) RevokeHash(ctx context.Context, recordID string, version int) (CommitResult, error) {
	if s.revoke != nil {
		return s.revoke(ctx, recordID, version)
	}
	return CommitResult{}, nil
}

func (s chainStub) Verify(ctx context.Context, recordID, dataHash string) (bool, error) {
	return s.verify(ctx, recordID, dataHash)
}

func (s chainStub) GetLatest(ctx context.Context, recordID string) (LatestRecord, error) {
	return s.latest(ctx, recordID)
}

func (s chainStub) Receipt(ctx context.Context, txHash string) (CommitResult, error) {
	if s.receipt != nil {
		return s.receipt(ctx, txHash)
	}
	return CommitResult{TxHash: txHash, Mined: false}, nil
}

type auditStub struct {
	log func(ctx context.Context, input audit.Input) error
}

func (s auditStub) Log(ctx context.Context, input audit.Input) error {
	if s.log != nil {
		return s.log(ctx, input)
	}
	return nil
}

type notifierStub struct {
	notify func(ctx context.Context, payload IntegrityAlertPayload) error
	hits   int
}

func (s *notifierStub) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityAlertPayload) error {
	s.hits++
	if s.notify != nil {
		return s.notify(ctx, payload)
	}
	return nil
}

func TestPreparePledgeSetsHashAndStatuses(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	service := NewService(nil, nil, nil)
	pledge, err := service.PreparePledge(domain.Pledge{
		PledgeID:        "pledge-1",
		ShopID:          "shop-1",
		CreatedByUserID: "user-1",
		Status:          "committed",
		Version:         1,
		Score:           8.5,
		Category:        "fresh_produce",
		Confidence:      0.91,
		ImageHash:       "img-hash",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.DataHash == "" || pledge.ChainAnchorStatus != ChainAnchorStatusPending || pledge.IntegrityStatus != IntegrityStatusPendingAnchor {
		t.Fatalf("unexpected prepared pledge: %#v", pledge)
	}
}

func TestSyncPledgeAnchorsMinedCommit(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	service := NewService(nil, chainStub{
		commit: func(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (CommitResult, error) {
			if recordID != "pledge-1" || dataHash == "" || version != 1 {
				t.Fatalf("unexpected commit input: %s %s %d", recordID, dataHash, version)
			}
			return CommitResult{
				TxHash:      "0xtx1",
				BlockNumber: 12,
				BlockTime:   &now,
				Mined:       true,
			}, nil
		},
	}, nil)

	pledge, err := service.SyncPledge(context.Background(), domain.Pledge{
		PledgeID:        "pledge-1",
		ShopID:          "shop-1",
		CreatedByUserID: "user-1",
		Status:          "committed",
		Version:         1,
		Score:           8.5,
		Category:        "fresh_produce",
		Confidence:      0.91,
		ImageHash:       "img-hash",
		DataHash:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.ChainTxHash != "0xtx1" || pledge.ChainBlockNumber != 12 || pledge.ChainAnchorStatus != ChainAnchorStatusAnchored || pledge.IntegrityStatus != IntegrityStatusAnchored {
		t.Fatalf("unexpected anchored pledge: %#v", pledge)
	}
}

func TestVerifyAnchoredPledgesMarksMismatch(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	saved := domain.Pledge{}
	notifier := &notifierStub{
		notify: func(ctx context.Context, payload IntegrityAlertPayload) error {
			if payload.PledgeID != "pledge-1" || payload.OnChainDataHash == "" || payload.IntegrityStatus != IntegrityStatusMismatchDetected {
				t.Fatalf("unexpected alert payload: %#v", payload)
			}
			return nil
		},
	}
	service := NewService(pledgeRepositoryStub{
		listByChainAnchorStatus: func(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
			return []domain.Pledge{{
				PledgeID:          "pledge-1",
				ShopID:            "shop-1",
				CreatedByUserID:   "user-1",
				Status:            "committed",
				Version:           1,
				DataHash:          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ChainAnchorStatus: ChainAnchorStatusAnchored,
				IntegrityStatus:   IntegrityStatusAnchored,
				CreatedAt:         now,
				UpdatedAt:         now,
			}}, nil
		},
		save: func(ctx context.Context, pledge domain.Pledge) error {
			saved = pledge
			return nil
		},
	}, chainStub{
		verify: func(ctx context.Context, recordID, dataHash string) (bool, error) {
			return false, nil
		},
		latest: func(ctx context.Context, recordID string) (LatestRecord, error) {
			return LatestRecord{
				DataHash:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Version:   1,
				IsPresent: true,
			}, nil
		},
	}, auditStub{})
	service.SetNotifier(notifier)
	service.now = func() time.Time { return now.Add(time.Minute) }

	if err := service.VerifyAnchoredPledges(context.Background(), 10); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if saved.IntegrityStatus != IntegrityStatusMismatchDetected {
		t.Fatalf("expected mismatch status, got %#v", saved)
	}
	if notifier.hits != 1 {
		t.Fatalf("expected one notifier hit, got %d", notifier.hits)
	}
}

func TestRevokePledgeUpdatesIntegrityStatus(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	service := NewService(nil, chainStub{
		revoke: func(ctx context.Context, recordID string, version int) (CommitResult, error) {
			if recordID != "pledge-1" || version != 2 {
				t.Fatalf("unexpected revoke input: %s %d", recordID, version)
			}
			return CommitResult{TxHash: "0xrevoke", BlockNumber: 22, BlockTime: &now, Mined: true}, nil
		},
	}, nil)

	pledge, err := service.RevokePledge(context.Background(), domain.Pledge{
		PledgeID:          "pledge-1",
		Version:           1,
		ChainAnchorStatus: ChainAnchorStatusAnchored,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.Version != 2 || pledge.IntegrityStatus != IntegrityStatusRevoked || pledge.ChainTxHash != "0xrevoke" {
		t.Fatalf("unexpected revoked pledge: %#v", pledge)
	}
}
