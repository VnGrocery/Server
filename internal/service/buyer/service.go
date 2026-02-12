package buyer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	visionservice "vngrocery/internal/service/vision"
)

var ErrInvalidCheck = errors.New("invalid buyer check request")

type CheckInput struct {
	PledgeID string
	Image    visionservice.ImageInput
}

type CheckResult struct {
	PolicyVersion    string
	PledgeID         string
	Trusted          bool
	Verdict          string
	PledgedScore     float64
	ActualScore      float64
	ScoreDelta       float64
	ScoreDeltaAbs    float64
	PledgedCategory  string
	ActualCategory   string
	ActualConfidence float64
	CategoryMatch    bool
	Reasons          []string
}

type CheckService interface {
	Check(ctx context.Context, input CheckInput) (CheckResult, error)
}

type Service struct {
	pledges repository.PledgeRepository
	scorer  visionservice.ImageScorer
}

func NewService(pledges repository.PledgeRepository, scorer visionservice.ImageScorer) *Service {
	return &Service{
		pledges: pledges,
		scorer:  scorer,
	}
}

func (s *Service) Check(ctx context.Context, input CheckInput) (CheckResult, error) {
	if strings.TrimSpace(input.PledgeID) == "" {
		return CheckResult{}, fmt.Errorf("%w: pledgeId is required", ErrInvalidCheck)
	}
	if s.pledges == nil {
		return CheckResult{}, fmt.Errorf("pledge repository is not configured")
	}
	if s.scorer == nil {
		return CheckResult{}, visionservice.ErrProviderUnavailable
	}

	pledge, err := s.pledges.GetByID(ctx, strings.TrimSpace(input.PledgeID))
	if err != nil {
		return CheckResult{}, err
	}

	scored, err := s.scorer.Score(ctx, input.Image)
	if err != nil {
		return CheckResult{}, err
	}

	return comparePledge(pledge, scored), nil
}

const (
	policyVersionV1       = "trust_policy_v1"
	trustedMaxScoreDelta  = 1.0
	warningMaxScoreDelta  = 2.5
	minRequiredConfidence = 0.60
)

func comparePledge(pledge domain.Pledge, scored visionservice.ScoreResult) CheckResult {
	scoreDelta := scored.Score - pledge.Score
	absoluteDelta := math.Abs(scoreDelta)
	categoryMatch := strings.EqualFold(strings.TrimSpace(pledge.Category), strings.TrimSpace(scored.Category))
	confidenceEnough := scored.Confidence >= minRequiredConfidence

	reasons := make([]string, 0, 2)
	verdict := "warning"
	trusted := false

	if !categoryMatch {
		reasons = append(reasons, "category_mismatch")
	}
	if !confidenceEnough {
		reasons = append(reasons, "low_ai_confidence")
	}

	switch {
	case categoryMatch && confidenceEnough && absoluteDelta <= trustedMaxScoreDelta:
		trusted = true
		verdict = "trusted"
	case !categoryMatch || absoluteDelta > warningMaxScoreDelta:
		verdict = "high_risk"
		if absoluteDelta > warningMaxScoreDelta {
			reasons = append(reasons, "score_gap_high")
		}
	default:
		reasons = append(reasons, "score_gap_warning")
	}

	return CheckResult{
		PolicyVersion:    policyVersionV1,
		PledgeID:         pledge.PledgeID,
		Trusted:          trusted,
		Verdict:          verdict,
		PledgedScore:     pledge.Score,
		ActualScore:      scored.Score,
		ScoreDelta:       scoreDelta,
		ScoreDeltaAbs:    absoluteDelta,
		PledgedCategory:  pledge.Category,
		ActualCategory:   scored.Category,
		ActualConfidence: scored.Confidence,
		CategoryMatch:    categoryMatch,
		Reasons:          reasons,
	}
}
