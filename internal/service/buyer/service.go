package buyer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

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
	CheckID          string
	ShopID           string
	PolicyVersion    string
	HasPledge        bool
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
	checks  repository.BuyerCheckRepository
	scorer  visionservice.ImageScorer
	now     func() time.Time
}

func NewService(pledges repository.PledgeRepository, checks repository.BuyerCheckRepository, scorer visionservice.ImageScorer) *Service {
	return &Service{
		pledges: pledges,
		checks:  checks,
		scorer:  scorer,
		now:     time.Now,
	}
}

func (s *Service) Check(ctx context.Context, input CheckInput) (CheckResult, error) {
	if s.scorer == nil {
		return CheckResult{}, visionservice.ErrProviderUnavailable
	}

	pledgeID := strings.TrimSpace(input.PledgeID)
	var pledge domain.Pledge
	if s.pledges == nil {
		if pledgeID != "" {
			return CheckResult{}, fmt.Errorf("pledge repository is not configured")
		}
	} else if pledgeID != "" {
		var err error
		pledge, err = s.pledges.GetByID(ctx, pledgeID)
		if err != nil {
			return CheckResult{}, err
		}
	}

	scored, err := s.scorer.Score(ctx, input.Image)
	if err != nil {
		return CheckResult{}, err
	}

	if pledgeID == "" {
		return standaloneQualityResult(scored), nil
	}

	result := comparePledge(pledge, scored)
	if s.checks != nil {
		check := result.toBuyerCheck(uuid.NewString(), s.now().UTC())
		if err := s.checks.Save(ctx, check); err != nil {
			return CheckResult{}, err
		}
		result.CheckID = check.CheckID
	}

	return result, nil
}

const (
	policyVersionV1       = "trust_policy_v1"
	trustedMaxScoreDelta  = 1.0
	warningMaxScoreDelta  = 2.5
	minRequiredConfidence = 0.60
)

func standaloneQualityResult(scored visionservice.ScoreResult) CheckResult {
	return CheckResult{
		PolicyVersion:    policyVersionV1,
		HasPledge:        false,
		Trusted:          false,
		Verdict:          "no_pledge",
		ActualScore:      scored.Score,
		ActualCategory:   scored.Category,
		ActualConfidence: scored.Confidence,
		CategoryMatch:    false,
		Reasons:          []string{"no_seller_pledge"},
	}
}

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
		ShopID:           pledge.ShopID,
		PolicyVersion:    policyVersionV1,
		HasPledge:        true,
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

func (r CheckResult) toBuyerCheck(checkID string, createdAt time.Time) domain.BuyerCheck {
	return domain.BuyerCheck{
		CheckID:          checkID,
		ShopID:           r.ShopID,
		PledgeID:         r.PledgeID,
		Status:           "completed",
		PolicyVersion:    r.PolicyVersion,
		Trusted:          r.Trusted,
		Verdict:          r.Verdict,
		PledgedScore:     r.PledgedScore,
		ActualScore:      r.ActualScore,
		ScoreDelta:       r.ScoreDelta,
		ScoreDeltaAbs:    r.ScoreDeltaAbs,
		PledgedCategory:  r.PledgedCategory,
		ActualCategory:   r.ActualCategory,
		ActualConfidence: r.ActualConfidence,
		CategoryMatch:    r.CategoryMatch,
		Reasons:          r.Reasons,
		CreatedAt:        createdAt,
	}
}
