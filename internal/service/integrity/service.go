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
	ChainAnchorStatusPending   = "pending_anchor"
	ChainAnchorStatusAnchored  = "anchored"
	ChainAnchorOperationCommit = "commit"
	ChainAnchorOperationRevoke = "revoke"

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
	pledges     repository.PledgeRepository
	shops       repository.ShopRepository
	engagements repository.EngagementRepository
	chain       ChainClient
	audit       AuditLogger
	notifier    Notifier
	observer    Observer
	now         func() time.Time
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
	PledgeID        string  `json:"pledgeId"`
	ShopID          string  `json:"shopId"`
	ProductID       string  `json:"productId,omitempty"`
	CreatedByUserID string  `json:"createdByUserId"`
	Status          string  `json:"status"`
	Version         int     `json:"version"`
	Score           float64 `json:"score"`
	Category        string  `json:"category"`
	Confidence      float64 `json:"confidence"`
	ImageHash       string  `json:"imageHash"`
	ImageCID        string  `json:"imageCid,omitempty"`

	// Why the seller recorded this score. Inside the hash that gets anchored,
	// so it cannot be rewritten afterwards - a note beside the chain would be
	// a caption, not evidence. omitempty keeps the pledges anchored before this
	// field existed hashing exactly as they did.
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type shopHashPayload struct {
	ShopID      string    `json:"shopId"`
	OwnerUserID string    `json:"ownerUserId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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

func (s *Service) SetShopRepository(shops repository.ShopRepository) {
	s.shops = shops
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
	pledge.ChainAnchorOperation = ChainAnchorOperationCommit
	pledge.ChainAnchorAttempts = 0
	pledge.ChainAnchorNextAttemptAt = nil
	pledge.ChainAnchorLastError = ""
	return pledge, nil
}

func (s *Service) SyncPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if s.chain == nil {
		return pledge, nil
	}
	if pledge.ChainAnchorNextAttemptAt != nil && s.now().UTC().Before(*pledge.ChainAnchorNextAttemptAt) {
		return pledge, nil
	}
	if pledge.ChainAnchorOperation == ChainAnchorOperationRevoke {
		return s.syncPledgeRevoke(ctx, pledge)
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
		return s.markPledgeRetry(pledge, err), err
	}
	if reconciled, ok := s.reconcileCommittedPledge(ctx, pledge); ok {
		return reconciled, nil
	}

	if s.observer != nil {
		s.observer.IncIntegrityAnchorAttempt()
	}
	commit, err := s.chain.CommitHash(ctx, pledge.PledgeID, pledge.DataHash, pledge.CreatedAt, pledge.Version)
	if err != nil {
		if commit.TxHash != "" {
			pledge.ChainTxHash = commit.TxHash
		}
		if s.observer != nil {
			s.observer.IncIntegrityAnchorFailure()
		}
		return s.markPledgeRetry(pledge, err), err
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

func (s *Service) PrepareShop(shop domain.Shop) (domain.Shop, error) {
	hash, err := HashShop(shop)
	if err != nil {
		return domain.Shop{}, err
	}

	shop.DataHash = hash
	shop.ChainAnchorStatus = ChainAnchorStatusPending
	shop.IntegrityStatus = IntegrityStatusPendingAnchor
	shop.ChainTxHash = ""
	shop.ChainBlockNumber = 0
	shop.ChainAnchorTime = nil
	shop.ChainAnchorOperation = ChainAnchorOperationCommit
	shop.ChainAnchorAttempts = 0
	shop.ChainAnchorNextAttemptAt = nil
	shop.ChainAnchorLastError = ""
	return shop, nil
}

func (s *Service) SyncShop(ctx context.Context, shop domain.Shop) (domain.Shop, error) {
	if s.chain == nil {
		return shop, nil
	}
	if strings.TrimSpace(shop.DataHash) == "" {
		var err error
		shop, err = s.PrepareShop(shop)
		if err != nil {
			return domain.Shop{}, err
		}
	}
	if shop.ChainAnchorStatus == ChainAnchorStatusAnchored {
		return shop, nil
	}
	if shop.ChainAnchorNextAttemptAt != nil && s.now().UTC().Before(*shop.ChainAnchorNextAttemptAt) {
		return shop, nil
	}
	if strings.TrimSpace(shop.ChainTxHash) != "" {
		receipt, err := s.chain.Receipt(ctx, shop.ChainTxHash)
		if err == nil && receipt.Mined {
			return s.applyAnchoredShop(shop, receipt), nil
		}
		if err == nil {
			return shop, nil
		}
		return s.markShopRetry(shop, err), err
	}
	if reconciled, ok := s.reconcileCommittedShop(ctx, shop); ok {
		return reconciled, nil
	}

	commit, err := s.chain.CommitHash(ctx, shopRecordID(shop.ShopID), shop.DataHash, shop.CreatedAt, shop.Version)
	if err != nil {
		if commit.TxHash != "" {
			shop.ChainTxHash = commit.TxHash
		}
		return s.markShopRetry(shop, err), err
	}
	shop.ChainTxHash = commit.TxHash
	if commit.Mined {
		return s.applyAnchoredShop(shop, commit), nil
	}
	return shop, nil
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
			if saveErr := s.pledges.Save(ctx, updated); saveErr != nil {
				return saveErr
			}
			continue
		}
		if err := s.pledges.Save(ctx, updated); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ProcessPendingShops(ctx context.Context, limit int) error {
	if s.shops == nil || s.chain == nil {
		return nil
	}
	shops, err := s.shops.List(ctx, repository.ShopListFilter{})
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = 25
	}
	processed := 0
	for _, shop := range shops {
		if strings.TrimSpace(shop.DataHash) == "" && strings.TrimSpace(shop.ChainAnchorStatus) == "" {
			continue
		}
		if shop.ChainAnchorStatus != ChainAnchorStatusPending {
			continue
		}
		updated, err := s.SyncShop(ctx, shop)
		if err != nil {
			if saveErr := s.shops.Save(ctx, updated); saveErr != nil {
				return saveErr
			}
			continue
		}
		if err := s.shops.Save(ctx, updated); err != nil {
			return err
		}
		processed++
		if processed >= limit {
			break
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
		view.ProvidedHashMatch = strings.EqualFold(view.ProvidedDataHash, strings.TrimSpace(strings.TrimPrefix(pledge.DataHash, "0x")))
		if !view.ProvidedHashMatch && view.MismatchReason == "" {
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
		ImageCID:        strings.TrimSpace(pledge.ImageCID),
		Note:            strings.TrimSpace(pledge.Note),
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

func HashShop(shop domain.Shop) (string, error) {
	payload := shopHashPayload{
		ShopID:      strings.TrimSpace(shop.ShopID),
		OwnerUserID: strings.TrimSpace(shop.OwnerUserID),
		Name:        strings.TrimSpace(shop.Name),
		Description: strings.TrimSpace(shop.Description),
		Address:     strings.TrimSpace(shop.Address),
		Latitude:    shop.Latitude,
		Longitude:   shop.Longitude,
		Status:      strings.TrimSpace(shop.Status),
		Version:     shop.Version,
		CreatedAt:   shop.CreatedAt.UTC(),
		UpdatedAt:   shop.UpdatedAt.UTC(),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal shop payload: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) verifyPledge(ctx context.Context, pledge domain.Pledge) (bool, LatestRecord, error) {
	latest, err := s.chain.GetLatest(ctx, pledge.PledgeID)
	if err != nil {
		return false, LatestRecord{}, err
	}
	if !latest.IsPresent {
		return false, latest, nil
	}
	if latest.IsRevoked {
		return false, latest, nil
	}
	ok := strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(pledge.DataHash, "0x")), strings.TrimSpace(strings.TrimPrefix(latest.DataHash, "0x")))
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
	pledge.Version = nextVersion
	pledge.IntegrityStatus = IntegrityStatusPendingAnchor
	pledge.ChainAnchorStatus = ChainAnchorStatusPending
	pledge.ChainAnchorOperation = ChainAnchorOperationRevoke
	pledge.ChainTxHash = ""
	pledge.ChainBlockNumber = 0
	pledge.ChainAnchorTime = nil
	pledge.ChainAnchorAttempts = 0
	pledge.ChainAnchorNextAttemptAt = nil
	pledge.ChainAnchorLastError = ""
	pledge.UpdatedAt = s.now().UTC()
	return pledge, nil
}

func (s *Service) ReanchorPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if s.chain == nil {
		return domain.Pledge{}, fmt.Errorf("integrity chain client is not configured")
	}
	if s.observer != nil {
		s.observer.IncIntegrityReanchor()
	}
	pledge.Version++
	pledge.UpdatedAt = s.now().UTC()
	rehashed, err := HashPledge(pledge)
	if err != nil {
		return domain.Pledge{}, err
	}
	pledge.DataHash = rehashed
	pledge.ChainAnchorStatus = ChainAnchorStatusPending
	pledge.IntegrityStatus = IntegrityStatusReanchored
	pledge.ChainTxHash = ""
	pledge.ChainBlockNumber = 0
	pledge.ChainAnchorTime = nil
	pledge.ChainAnchorOperation = ChainAnchorOperationCommit
	pledge.ChainAnchorAttempts = 0
	pledge.ChainAnchorNextAttemptAt = nil
	pledge.ChainAnchorLastError = ""
	return pledge, nil
}

func (s *Service) syncPledgeRevoke(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	if strings.TrimSpace(pledge.ChainTxHash) != "" {
		receipt, err := s.chain.Receipt(ctx, pledge.ChainTxHash)
		if err == nil && receipt.Mined {
			pledge = s.applyAnchored(pledge, receipt)
			pledge.IntegrityStatus = IntegrityStatusRevoked
			return pledge, nil
		}
		if err == nil {
			return pledge, nil
		}
		return s.markPledgeRetry(pledge, err), err
	}
	if latest, err := s.chain.GetLatest(ctx, pledge.PledgeID); err == nil && latest.IsPresent && latest.IsRevoked && latest.Version == pledge.Version {
		pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
		pledge.IntegrityStatus = IntegrityStatusRevoked
		pledge.ChainAnchorTime = latest.Timestamp
		pledge.ChainAnchorOperation = ""
		pledge.ChainAnchorAttempts = 0
		pledge.ChainAnchorNextAttemptAt = nil
		pledge.ChainAnchorLastError = ""
		return pledge, nil
	}
	commit, err := s.chain.RevokeHash(ctx, pledge.PledgeID, pledge.Version)
	if err != nil {
		if commit.TxHash != "" {
			pledge.ChainTxHash = commit.TxHash
		}
		return s.markPledgeRetry(pledge, err), err
	}
	pledge.ChainTxHash = commit.TxHash
	if commit.Mined {
		pledge = s.applyAnchored(pledge, commit)
		pledge.IntegrityStatus = IntegrityStatusRevoked
	}
	return pledge, nil
}

func (s *Service) reconcileCommittedPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, bool) {
	latest, err := s.chain.GetLatest(ctx, pledge.PledgeID)
	if err != nil || !latest.IsPresent || latest.IsRevoked || latest.Version != pledge.Version || !sameHash(latest.DataHash, pledge.DataHash) {
		return pledge, false
	}
	pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
	if pledge.IntegrityStatus != IntegrityStatusReanchored {
		pledge.IntegrityStatus = IntegrityStatusAnchored
	}
	pledge.ChainAnchorTime = latest.Timestamp
	pledge.ChainAnchorOperation = ""
	pledge.ChainAnchorAttempts = 0
	pledge.ChainAnchorNextAttemptAt = nil
	pledge.ChainAnchorLastError = ""
	return pledge, true
}

func (s *Service) reconcileCommittedShop(ctx context.Context, shop domain.Shop) (domain.Shop, bool) {
	latest, err := s.chain.GetLatest(ctx, shopRecordID(shop.ShopID))
	if err != nil || !latest.IsPresent || latest.IsRevoked || latest.Version != shop.Version || !sameHash(latest.DataHash, shop.DataHash) {
		return shop, false
	}
	shop.ChainAnchorStatus = ChainAnchorStatusAnchored
	shop.IntegrityStatus = IntegrityStatusAnchored
	shop.ChainAnchorTime = latest.Timestamp
	shop.ChainAnchorOperation = ""
	shop.ChainAnchorAttempts = 0
	shop.ChainAnchorNextAttemptAt = nil
	shop.ChainAnchorLastError = ""
	return shop, true
}

func (s *Service) markPledgeRetry(pledge domain.Pledge, cause error) domain.Pledge {
	pledge.ChainAnchorAttempts++
	next := s.now().UTC().Add(anchorRetryDelay(pledge.ChainAnchorAttempts))
	pledge.ChainAnchorNextAttemptAt = &next
	pledge.ChainAnchorLastError = compactError(cause)
	return pledge
}

func (s *Service) markShopRetry(shop domain.Shop, cause error) domain.Shop {
	shop.ChainAnchorAttempts++
	next := s.now().UTC().Add(anchorRetryDelay(shop.ChainAnchorAttempts))
	shop.ChainAnchorNextAttemptAt = &next
	shop.ChainAnchorLastError = compactError(cause)
	return shop
}

func anchorRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 6 {
		shift = 6
	}
	delay := 5 * time.Second * time.Duration(1<<shift)
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimSpace(err.Error())
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

func sameHash(left, right string) bool {
	return strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(left), "0x"), strings.TrimPrefix(strings.TrimSpace(right), "0x"))
}

func (s *Service) applyAnchored(pledge domain.Pledge, commit CommitResult) domain.Pledge {
	pledge.ChainTxHash = commit.TxHash
	pledge.ChainBlockNumber = commit.BlockNumber
	pledge.ChainAnchorStatus = ChainAnchorStatusAnchored
	pledge.ChainAnchorOperation = ""
	pledge.ChainAnchorAttempts = 0
	pledge.ChainAnchorNextAttemptAt = nil
	pledge.ChainAnchorLastError = ""
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

func (s *Service) applyAnchoredShop(shop domain.Shop, commit CommitResult) domain.Shop {
	shop.ChainTxHash = commit.TxHash
	shop.ChainBlockNumber = commit.BlockNumber
	shop.ChainAnchorStatus = ChainAnchorStatusAnchored
	shop.ChainAnchorOperation = ""
	shop.ChainAnchorAttempts = 0
	shop.ChainAnchorNextAttemptAt = nil
	shop.ChainAnchorLastError = ""
	shop.IntegrityStatus = IntegrityStatusAnchored
	if commit.BlockTime != nil {
		shop.ChainAnchorTime = commit.BlockTime
	} else {
		now := s.now().UTC()
		shop.ChainAnchorTime = &now
	}
	return shop
}

func shopRecordID(shopID string) string {
	return "shop:" + strings.TrimSpace(shopID)
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
