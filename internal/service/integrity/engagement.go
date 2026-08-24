package integrity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

// Anchoring for engagement counts, kept beside the pledge and shop paths
// rather than folded into them: a count is only ever committed and re-committed
// as it moves, never revoked and never disputed against a submitted hash, so it
// needs neither the revoke branch nor the mismatch alerting those two carry.

type engagementCountHashPayload struct {
	CountID    string `json:"countId"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Follows    int    `json:"follows"`
	Likes      int    `json:"likes"`
	Loves      int    `json:"loves"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updatedAt"`
}

func HashEngagementCount(count domain.EngagementCount) (string, error) {
	raw, err := json.Marshal(engagementCountHashPayload{
		CountID:    strings.TrimSpace(count.CountID),
		TargetType: strings.TrimSpace(count.TargetType),
		TargetID:   strings.TrimSpace(count.TargetID),
		Follows:    count.Follows,
		Likes:      count.Likes,
		Loves:      count.Loves,
		Version:    count.Version,
		UpdatedAt:  count.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal engagement count payload: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) SetEngagementRepository(engagements repository.EngagementRepository) {
	s.engagements = engagements
}

// PrepareEngagementCount marks the new figure as owing an anchor. The worker
// picks it up on its next tick, so a burst of hearts on one product collapses
// into a single transaction carrying the total they add up to.
func (s *Service) PrepareEngagementCount(count domain.EngagementCount) (domain.EngagementCount, error) {
	hash, err := HashEngagementCount(count)
	if err != nil {
		return domain.EngagementCount{}, err
	}

	count.DataHash = hash
	count.ChainAnchorStatus = ChainAnchorStatusPending
	count.IntegrityStatus = IntegrityStatusPendingAnchor
	count.ChainTxHash = ""
	count.ChainBlockNumber = 0
	count.ChainAnchorTime = nil
	count.ChainAnchorAttempts = 0
	count.ChainAnchorNextAttemptAt = nil
	count.ChainAnchorLastError = ""
	return count, nil
}

func (s *Service) SyncEngagementCount(ctx context.Context, count domain.EngagementCount) (domain.EngagementCount, error) {
	if s.chain == nil {
		return count, nil
	}
	if strings.TrimSpace(count.DataHash) == "" {
		var err error
		count, err = s.PrepareEngagementCount(count)
		if err != nil {
			return domain.EngagementCount{}, err
		}
	}
	if count.ChainAnchorStatus == ChainAnchorStatusAnchored {
		return count, nil
	}
	if count.ChainAnchorNextAttemptAt != nil && s.now().UTC().Before(*count.ChainAnchorNextAttemptAt) {
		return count, nil
	}
	if strings.TrimSpace(count.ChainTxHash) != "" {
		receipt, err := s.chain.Receipt(ctx, count.ChainTxHash)
		if err == nil && receipt.Mined {
			return s.applyAnchoredEngagementCount(count, receipt), nil
		}
		if err == nil {
			return count, nil
		}
		return s.markEngagementCountRetry(count, err), err
	}

	commit, err := s.chain.CommitHash(ctx, engagementRecordID(count.CountID), count.DataHash, count.UpdatedAt, count.Version)
	if err != nil {
		if commit.TxHash != "" {
			count.ChainTxHash = commit.TxHash
		}
		return s.markEngagementCountRetry(count, err), err
	}
	count.ChainTxHash = commit.TxHash
	if commit.Mined {
		return s.applyAnchoredEngagementCount(count, commit), nil
	}
	return count, nil
}

func (s *Service) ProcessPendingEngagementCounts(ctx context.Context, limit int) error {
	if s.engagements == nil || s.chain == nil {
		return nil
	}
	counts, err := s.engagements.ListCountsByChainAnchorStatus(ctx, ChainAnchorStatusPending, limit)
	if err != nil {
		return err
	}
	for _, count := range counts {
		updated, syncErr := s.SyncEngagementCount(ctx, count)
		if saveErr := s.engagements.SaveCount(ctx, updated); saveErr != nil {
			return saveErr
		}
		if syncErr != nil {
			continue
		}
	}
	return nil
}

// VerifyEngagementCount asks the chain whether the figure being served is the
// one that was anchored. A count nobody has anchored yet answers false without
// that being a fault.
func (s *Service) VerifyEngagementCount(ctx context.Context, count domain.EngagementCount) (bool, error) {
	if s.chain == nil || strings.TrimSpace(count.DataHash) == "" {
		return false, nil
	}
	return s.chain.Verify(ctx, engagementRecordID(count.CountID), count.DataHash)
}

func (s *Service) applyAnchoredEngagementCount(count domain.EngagementCount, commit CommitResult) domain.EngagementCount {
	count.ChainTxHash = commit.TxHash
	count.ChainBlockNumber = commit.BlockNumber
	count.ChainAnchorStatus = ChainAnchorStatusAnchored
	count.IntegrityStatus = IntegrityStatusAnchored
	count.ChainAnchorAttempts = 0
	count.ChainAnchorNextAttemptAt = nil
	count.ChainAnchorLastError = ""
	if commit.BlockTime != nil {
		count.ChainAnchorTime = commit.BlockTime
	} else {
		now := s.now().UTC()
		count.ChainAnchorTime = &now
	}
	return count
}

func (s *Service) markEngagementCountRetry(count domain.EngagementCount, cause error) domain.EngagementCount {
	count.ChainAnchorAttempts++
	next := s.now().UTC().Add(anchorRetryDelay(count.ChainAnchorAttempts))
	count.ChainAnchorNextAttemptAt = &next
	count.ChainAnchorLastError = compactError(cause)
	return count
}

func engagementRecordID(countID string) string {
	return "engagement:" + strings.TrimSpace(countID)
}
