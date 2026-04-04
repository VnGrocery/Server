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
	RevokeHash(ctx context.Context, recordID string, version int) (CommitResult, error)
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

type Notifier interface {
	NotifyIntegrityMismatch(ctx context.Context, payload IntegrityAlertPayload) error
}

type Service struct {
	pledges  repository.PledgeRepository
	chain    ChainClient
	audit    AuditLogger
	notifier Notifier
	observer Observer
	now      func() time.Time
}

type IntegrityView struct {
	PledgeID          string
	ShopID            string
	DataHash          string
	ProvidedDataHash  string
	ChainTxHash       string
	ChainBlockNumber  int64
	ChainAnchorStatus string
	ChainAnchorTime   *time.Time
	IntegrityStatus   string
	OnChainMatch      bool
	ProvidedHashMatch bool
	OnChainDataHash   string
	OnChainVersion    int
	OnChainTimestamp  *time.Time
	OnChainPresent    bool
	MismatchReason    string
	LastCheckedAt     *time.Time
	CanReanchor       bool
	CanRevoke         bool
}

type IntegrityAlertPayload struct {
	PledgeID         string
	ShopID           string
	CreatedByUserID  string
	DataHash         string
	ChainTxHash      string
	IntegrityStatus  string
	DetectedAt       time.Time
	OnChainDataHash  string
	OnChainVersion   int
	OnChainTimestamp *time.Time
}

type Observer interface {
	IncIntegrityAnchorAttempt()
	IncIntegrityAnchorSuccess()
	IncIntegrityAnchorFailure()
	IncIntegrityVerifyMismatch()
	IncIntegrityReanchor()
	IncIntegrityRevoke()
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

func (s *Service) SetNotifier(notifier Notifier) {
	s.notifier = notifier
}

func (s *Service) SetObserver(observer Observer) {
	s.observer = observer
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

	if s.observer != nil {
		s.observer.IncIntegrityAnchorAttempt()
	}
	commit, err := s.chain.CommitHash(ctx, pledge.PledgeID, pledge.DataHash, pledge.CreatedAt, pledge.Version)
	if err != nil {
		if s.observer != nil {
			s.observer.IncIntegrityAnchorFailure()
		}
		return pledge, err
	}

	pledge.ChainTxHash = commit.TxHash
	if commit.Mined {
		if s.observer != nil {
			s.observer.IncIntegrityAnchorSuccess()
		}
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
		if s.observer != nil {
			s.observer.IncIntegrityVerifyMismatch()
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
		if s.notifier != nil {
			_ = s.notifier.NotifyIntegrityMismatch(ctx, IntegrityAlertPayload{
				PledgeID:         pledge.PledgeID,
				ShopID:           pledge.ShopID,
				CreatedByUserID:  pledge.CreatedByUserID,
				DataHash:         pledge.DataHash,
				ChainTxHash:      pledge.ChainTxHash,
				IntegrityStatus:  pledge.IntegrityStatus,
				DetectedAt:       pledge.UpdatedAt,
				OnChainDataHash:  latest.DataHash,
				OnChainVersion:   latest.Version,
				OnChainTimestamp: latest.Timestamp,
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
		LastCheckedAt:     pointerTime(pledge.UpdatedAt),
		CanReanchor:       pledge.IntegrityStatus == IntegrityStatusMismatchDetected || pledge.IntegrityStatus == IntegrityStatusRevoked,
		CanRevoke:         pledge.ChainAnchorStatus == ChainAnchorStatusAnchored && pledge.IntegrityStatus != IntegrityStatusRevoked,
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
	view.OnChainPresent = latest.IsPresent
	view.MismatchReason = mismatchReason(pledge, latest, ok)
	return view, nil
}

func (s *Service) VerifyPledgeHash(ctx context.Context, pledge domain.Pledge, dataHash string) (IntegrityView, error) {
	view, err := s.GetPledgeIntegrity(ctx, pledge)
	if err != nil {
		return IntegrityView{}, err
	}
	view.ProvidedDataHash = strings.TrimSpace(strings.TrimPrefix(dataHash, "0x"))
	if view.ProvidedDataHash != "" {
		view.ProvidedHashMatch = strings.EqualFold(view.ProvidedDataHash, view.OnChainDataHash)
		if view.OnChainPresent && !view.ProvidedHashMatch && view.MismatchReason == "" {
			view.MismatchReason = "provided_hash_mismatch"
		}
	}
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

func (s *Service) RevokePledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if s.chain == nil {
		return domain.Pledge{}, fmt.Errorf("integrity chain client is not configured")
	}
	nextVersion := pledge.Version + 1
	if s.observer != nil {
		s.observer.IncIntegrityRevoke()
	}
	commit, err := s.chain.RevokeHash(ctx, pledge.PledgeID, nextVersion)
	if err != nil {
		return domain.Pledge{}, err
	}
	pledge.Version = nextVersion
	pledge.IntegrityStatus = IntegrityStatusRevoked
	pledge.ChainTxHash = commit.TxHash
	pledge.UpdatedAt = s.now().UTC()
	if commit.Mined {
		pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
		pledge.ChainBlockNumber = commit.BlockNumber
		pledge.ChainAnchorTime = commit.BlockTime
	}
	return pledge, nil
}

func (s *Service) ReanchorPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if s.chain == nil {
		return domain.Pledge{}, fmt.Errorf("integrity chain client is not configured")
	}
	if s.observer != nil {
		s.observer.IncIntegrityReanchor()
	}
	rehashed, err := HashPledge(pledge)
	if err != nil {
		return domain.Pledge{}, err
	}
	pledge.DataHash = rehashed
	pledge.Version++
	pledge.ChainAnchorStatus = ChainAnchorStatusPending
	pledge.IntegrityStatus = IntegrityStatusReanchored
	pledge.ChainTxHash = ""
	pledge.ChainBlockNumber = 0
	pledge.ChainAnchorTime = nil
	pledge.UpdatedAt = s.now().UTC()
	return s.SyncPledge(ctx, pledge)
}

func (s *Service) applyAnchored(pledge domain.Pledge, commit CommitResult) domain.Pledge {
	pledge.ChainTxHash = commit.TxHash
	pledge.ChainBlockNumber = commit.BlockNumber
	pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
	if pledge.IntegrityStatus != IntegrityStatusReanchored {
		pledge.IntegrityStatus = IntegrityStatusAnchored
	}
	if commit.BlockTime != nil {
		pledge.ChainAnchorTime = commit.BlockTime
	} else {
		now := s.now().UTC()
		pledge.ChainAnchorTime = &now
	}
	return pledge
}

func mismatchReason(pledge domain.Pledge, latest LatestRecord, matched bool) string {
	if matched {
		return ""
	}
	if !latest.IsPresent {
		return "missing_on_chain_record"
	}
	if latest.IsRevoked {
		return "revoked_on_chain"
	}
	if !strings.EqualFold(strings.TrimSpace(pledge.DataHash), strings.TrimSpace(latest.DataHash)) {
		return "data_hash_mismatch"
	}
	if latest.Version != pledge.Version {
		return "version_mismatch"
	}
	return "integrity_check_failed"
}

func pointerTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	v := value.UTC()
	return &v
}
