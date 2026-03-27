package integrity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

const (
	ChainAnchorStatusPending  = "pending_anchor"
	ChainAnchorStatusAnchored = "anchored"

	IntegrityStatusPendingAnchor    = "pending_anchor"
	IntegrityStatusAnchored         = "anchored"
	IntegrityStatusMismatchDetected = "mismatch_detected"
	IntegrityStatusRevoked          = "revoked"
	IntegrityStatusReanchored       = "reanchored"
)

type ChainClient interface {
	CommitHash(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (CommitResult, error)
	Verify(ctx context.Context, recordID, dataHash string) (bool, error)
	GetLatest(ctx context.Context, recordID string) (LatestRecord, error)
	Receipt(ctx context.Context, txHash string) (CommitResult, error)
}

type CommitResult struct {
	TxHash      string
	BlockNumber int64
	BlockTime   *time.Time
	Mined       bool
}

type LatestRecord struct {
	DataHash  string
	Timestamp *time.Time
	Version   int
	IsRevoked bool
	IsPresent bool
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	pledges repository.PledgeRepository
	chain   ChainClient
	audit   AuditLogger
	now     func() time.Time
}

type IntegrityView struct {
	PledgeID          string
	ShopID            string
	DataHash          string
	ChainTxHash       string
	ChainBlockNumber  int64
	ChainAnchorStatus string
	ChainAnchorTime   *time.Time
	IntegrityStatus   string
	OnChainMatch      bool
	OnChainDataHash   string
	OnChainVersion    int
	OnChainTimestamp  *time.Time
}

type pledgeHashPayload struct {
	PledgeID        string    `json:"pledgeId"`
	ShopID          string    `json:"shopId"`
	ProductID       string    `json:"productId,omitempty"`
	CreatedByUserID string    `json:"createdByUserId"`
	Status          string    `json:"status"`
	Version         int       `json:"version"`
	Score           float64   `json:"score"`
	Category        string    `json:"category"`
	Confidence      float64   `json:"confidence"`
	ImageHash       string    `json:"imageHash"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func NewService(pledges repository.PledgeRepository, chain ChainClient, auditLogger AuditLogger) *Service {
	return &Service{
		pledges: pledges,
		chain:   chain,
		audit:   auditLogger,
		now:     time.Now,
	}
}

func (s *Service) PreparePledge(pledge domain.Pledge) (domain.Pledge, error) {
	hash, err := HashPledge(pledge)
	if err != nil {
		return domain.Pledge{}, err
	}

	pledge.DataHash = hash
	pledge.ChainAnchorStatus = ChainAnchorStatusPending
	pledge.IntegrityStatus = IntegrityStatusPendingAnchor
	pledge.ChainTxHash = ""
	pledge.ChainBlockNumber = 0
	pledge.ChainAnchorTime = nil
	return pledge, nil
}

func (s *Service) SyncPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if s.chain == nil {
		return pledge, nil
	}
	if strings.TrimSpace(pledge.DataHash) == "" {
		var err error
		pledge, err = s.PreparePledge(pledge)
		if err != nil {
			return domain.Pledge{}, err
		}
	}

	if pledge.ChainAnchorStatus == ChainAnchorStatusAnchored {
		return pledge, nil
	}

	if strings.TrimSpace(pledge.ChainTxHash) != "" {
		receipt, err := s.chain.Receipt(ctx, pledge.ChainTxHash)
		if err == nil && receipt.Mined {
			return s.applyAnchored(pledge, receipt), nil
		}
		if err == nil {
			return pledge, nil
		}
	}

	commit, err := s.chain.CommitHash(ctx, pledge.PledgeID, pledge.DataHash, pledge.CreatedAt, pledge.Version)
	if err != nil {
		return pledge, err
	}

	pledge.ChainTxHash = commit.TxHash
	if commit.Mined {
		return s.applyAnchored(pledge, commit), nil
	}

	return pledge, nil
}

func (s *Service) ProcessPendingPledges(ctx context.Context, limit int) error {
	if s.pledges == nil || s.chain == nil {
		return nil
	}

	pledges, err := s.pledges.ListByChainAnchorStatus(ctx, ChainAnchorStatusPending, limit)
	if err != nil {
		return err
	}

	for _, pledge := range pledges {
		updated, err := s.SyncPledge(ctx, pledge)
		if err != nil {
			continue
		}
		if err := s.pledges.Save(ctx, updated); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) VerifyAnchoredPledges(ctx context.Context, limit int) error {
	if s.pledges == nil || s.chain == nil {
		return nil
	}

	pledges, err := s.pledges.ListByChainAnchorStatus(ctx, ChainAnchorStatusAnchored, limit)
	if err != nil {
		return err
	}

	for _, pledge := range pledges {
		ok, latest, err := s.verifyPledge(ctx, pledge)
		if err != nil {
			continue
		}
		if ok {
			continue
		}

		before := pledge
		pledge.IntegrityStatus = IntegrityStatusMismatchDetected
		if latest.IsRevoked {
			pledge.IntegrityStatus = IntegrityStatusRevoked
		}
		pledge.UpdatedAt = s.now().UTC()
		if err := s.pledges.Save(ctx, pledge); err != nil {
			return err
		}
		if s.audit != nil {
			_ = s.audit.Log(ctx, audit.Input{
				ActorUserID:     pledge.CreatedByUserID,
				ResourceType:    "pledge",
				ResourceID:      pledge.PledgeID,
				ResourceVersion: pledge.Version,
				Action:          "pledge.integrity_mismatch_detected",
				Status:          pledge.IntegrityStatus,
				Payload: audit.MutationPayload{
					Before: before,
					After:  pledge,
				},
			})
		}
	}

	return nil
}

func (s *Service) GetPledgeIntegrity(ctx context.Context, pledge domain.Pledge) (IntegrityView, error) {
	view := IntegrityView{
		PledgeID:          pledge.PledgeID,
		ShopID:            pledge.ShopID,
		DataHash:          pledge.DataHash,
		ChainTxHash:       pledge.ChainTxHash,
		ChainBlockNumber:  pledge.ChainBlockNumber,
		ChainAnchorStatus: pledge.ChainAnchorStatus,
		ChainAnchorTime:   pledge.ChainAnchorTime,
		IntegrityStatus:   pledge.IntegrityStatus,
	}

	if s.chain == nil || pledge.DataHash == "" {
		return view, nil
	}

	ok, latest, err := s.verifyPledge(ctx, pledge)
	if err != nil {
		return view, nil
	}
	view.OnChainMatch = ok
	view.OnChainDataHash = latest.DataHash
	view.OnChainVersion = latest.Version
	view.OnChainTimestamp = latest.Timestamp
	return view, nil
}

func HashPledge(pledge domain.Pledge) (string, error) {
	payload := pledgeHashPayload{
		PledgeID:        strings.TrimSpace(pledge.PledgeID),
		ShopID:          strings.TrimSpace(pledge.ShopID),
		ProductID:       strings.TrimSpace(pledge.ProductID),
		CreatedByUserID: strings.TrimSpace(pledge.CreatedByUserID),
		Status:          strings.TrimSpace(pledge.Status),
		Version:         pledge.Version,
		Score:           pledge.Score,
		Category:        strings.TrimSpace(pledge.Category),
		Confidence:      pledge.Confidence,
		ImageHash:       strings.TrimSpace(pledge.ImageHash),
		CreatedAt:       pledge.CreatedAt.UTC(),
		UpdatedAt:       pledge.UpdatedAt.UTC(),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pledge payload: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) verifyPledge(ctx context.Context, pledge domain.Pledge) (bool, LatestRecord, error) {
	ok, err := s.chain.Verify(ctx, pledge.PledgeID, pledge.DataHash)
	if err != nil {
		return false, LatestRecord{}, err
	}
	latest, err := s.chain.GetLatest(ctx, pledge.PledgeID)
	if err != nil {
		return ok, LatestRecord{}, err
	}
	if latest.IsPresent && latest.IsRevoked {
		return false, latest, nil
	}
	return ok, latest, nil
}

func (s *Service) applyAnchored(pledge domain.Pledge, commit CommitResult) domain.Pledge {
	pledge.ChainTxHash = commit.TxHash
	pledge.ChainBlockNumber = commit.BlockNumber
	pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
	pledge.IntegrityStatus = IntegrityStatusAnchored
	if commit.BlockTime != nil {
		pledge.ChainAnchorTime = commit.BlockTime
	} else {
		now := s.now().UTC()
		pledge.ChainAnchorTime = &now
	}
	return pledge
}
