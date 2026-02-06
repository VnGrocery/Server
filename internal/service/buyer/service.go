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
	PledgeID         string
	Trusted          bool
	Verdict          string
	PledgedScore     float64
	ActualScore      float64
	ScoreDelta       float64
	PledgedCategory  string
	ActualCategory   string
	ActualConfidence float64
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

func comparePledge(pledge domain.Pledge, scored visionservice.ScoreResult) CheckResult {
	scoreDelta := scored.Score - pledge.Score
	absoluteDelta := math.Abs(scoreDelta)
	categoryMatch := strings.EqualFold(strings.TrimSpace(pledge.Category), strings.TrimSpace(scored.Category))

	trusted := categoryMatch && absoluteDelta <= 1.5
	verdict := "warning"
	switch {
	case trusted:
		verdict = "trusted"
	case absoluteDelta > 3 || !categoryMatch:
		verdict = "high_risk"
	}

	return CheckResult{
		PledgeID:         pledge.PledgeID,
		Trusted:          trusted,
		Verdict:          verdict,
		PledgedScore:     pledge.Score,
		ActualScore:      scored.Score,
		ScoreDelta:       scoreDelta,
		PledgedCategory:  pledge.Category,
		ActualCategory:   scored.Category,
		ActualConfidence: scored.Confidence,
	}
}
