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
	"vngrocery/internal/service/audit"
	visionservice "vngrocery/internal/service/vision"
)

var ErrInvalidCheck = errors.New("invalid buyer check request")
var ErrRateLimited = errors.New("buyer check rate limit exceeded")

const (
	BuyerCheckStatusCompleted = "completed"
	BuyerCheckStatusFlagged   = "flagged"
	BuyerCheckStatusRejected  = "rejected"
)

type CheckInput struct {
	PledgeID    string
	BuyerUserID string
	ImageHash   string
	ImageCID    string
	Image       visionservice.ImageInput
}

type ModerateInput struct {
	CheckID         string
	ModeratorUserID string
	ExpectedVersion int
	Status          string
	ModerationNote  string
}

type ListInput struct {
	ActorUserID   string
	CheckID       string
	ShopID        string
	ProductID     string
	BuyerUserID   string
	Status        string
	Verdict       string
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Page          int
	PageSize      int
}

type ListResult struct {
	Items    []domain.BuyerCheck
	Page     int
	PageSize int
	Total    int
}

type CheckResult struct {
	CheckID          string
	ShopID           string
	ProductID        string
	PolicyVersion    string
	HasPledge        bool
	PledgeID         string
	BuyerUserID      string
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
	ImageHash        string
	ImageCID         string
	Reasons          []string
}

type CheckService interface {
	Check(ctx context.Context, input CheckInput) (CheckResult, error)
}

type Service struct {
	pledges repository.PledgeRepository
	checks  repository.BuyerCheckRepository
	users   repository.UserRepository
	scorer  visionservice.ImageScorer
	audit   AuditLogger
	now     func() time.Time
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

func NewService(pledges repository.PledgeRepository, checks repository.BuyerCheckRepository, users repository.UserRepository, scorer visionservice.ImageScorer, auditLogger AuditLogger) *Service {
	return &Service{
		pledges: pledges,
		checks:  checks,
		users:   users,
		scorer:  scorer,
		audit:   auditLogger,
		now:     time.Now,
	}
}

func (s *Service) Check(ctx context.Context, input CheckInput) (CheckResult, error) {
	buyerUserID := strings.TrimSpace(input.BuyerUserID)
	if buyerUserID == "" {
		return CheckResult{}, fmt.Errorf("%w: buyerUserId is required", ErrInvalidCheck)
	}
	if s.scorer == nil {
		return CheckResult{}, visionservice.ErrProviderUnavailable
	}
	if s.checks == nil {
		return CheckResult{}, fmt.Errorf("buyer check repository is not configured")
	}
	if err := s.ensureQuota(ctx, buyerUserID); err != nil {
		return CheckResult{}, err
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
		result := standaloneQualityResult(scored)
		result.BuyerUserID = buyerUserID
		result.ImageHash = strings.TrimSpace(input.ImageHash)
		result.ImageCID = strings.TrimSpace(input.ImageCID)
		return s.persistCheck(ctx, result)
	}

	result := comparePledge(pledge, scored)
	result.BuyerUserID = buyerUserID
	result.ImageHash = strings.TrimSpace(input.ImageHash)
	result.ImageCID = strings.TrimSpace(input.ImageCID)
	return s.persistCheck(ctx, result)
}

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (domain.BuyerCheck, error) {
	if strings.TrimSpace(input.CheckID) == "" {
		return domain.BuyerCheck{}, fmt.Errorf("%w: checkId is required", ErrInvalidCheck)
	}
	if strings.TrimSpace(input.ModeratorUserID) == "" {
		return domain.BuyerCheck{}, fmt.Errorf("%w: moderatorUserId is required", ErrInvalidCheck)
	}
	if input.ExpectedVersion <= 0 {
		return domain.BuyerCheck{}, fmt.Errorf("%w: expectedVersion must be positive", ErrInvalidCheck)
	}
	if s.checks == nil || s.users == nil {
		return domain.BuyerCheck{}, fmt.Errorf("buyer check moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ModeratorUserID); err != nil {
		return domain.BuyerCheck{}, err
	}

	check, err := s.checks.GetByID(ctx, strings.TrimSpace(input.CheckID))
	if err != nil || check.CheckID == "" {
		return domain.BuyerCheck{}, err
	}
	if check.Version != input.ExpectedVersion {
		return domain.BuyerCheck{}, fmt.Errorf("%w: version conflict", ErrInvalidCheck)
	}
	status, err := validateModerationStatus(input.Status)
	if err != nil {
		return domain.BuyerCheck{}, err
	}

	before := check
	check.Status = status
	check.Version++
	check.ModeratedByUserID = strings.TrimSpace(input.ModeratorUserID)
	check.ModerationNote = strings.TrimSpace(input.ModerationNote)
	now := s.now().UTC()
	check.ModeratedAt = &now
	check.UpdatedAt = now
	if err := s.checks.Save(ctx, check); err != nil {
		return domain.BuyerCheck{}, err
	}
	if s.audit != nil {
		if err := s.audit.Log(ctx, audit.Input{
			ActorUserID:     input.ModeratorUserID,
			ResourceType:    "buyer_check",
			ResourceID:      check.CheckID,
			ResourceVersion: check.Version,
			Action:          "buyer_check.moderated",
			Status:          check.Status,
			Payload: audit.MutationPayload{
				Before: before,
				After:  check,
			},
		}); err != nil {
			return domain.BuyerCheck{}, err
		}
	}

	return check, nil
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, error) {
	if strings.TrimSpace(input.ActorUserID) == "" {
		return ListResult{}, fmt.Errorf("%w: actorUserId is required", ErrInvalidCheck)
	}
	if s.checks == nil || s.users == nil {
		return ListResult{}, fmt.Errorf("buyer check list dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return ListResult{}, err
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	items, err := s.checks.List(ctx, repository.BuyerCheckListFilter{
		CheckID:       strings.TrimSpace(input.CheckID),
		ShopID:        strings.TrimSpace(input.ShopID),
		ProductID:     strings.TrimSpace(input.ProductID),
		BuyerUserID:   strings.TrimSpace(input.BuyerUserID),
		Status:        strings.TrimSpace(input.Status),
		Verdict:       strings.TrimSpace(input.Verdict),
		CreatedAfter:  input.CreatedAfter,
		CreatedBefore: input.CreatedBefore,
	})
	if err != nil {
		return ListResult{}, err
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return ListResult{Items: []domain.BuyerCheck{}, Page: page, PageSize: pageSize, Total: total}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return ListResult{
		Items:    items[start:end],
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
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
		ProductID:        pledge.ProductID,
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

func (s *Service) persistCheck(ctx context.Context, result CheckResult) (CheckResult, error) {
	check := result.toBuyerCheck(uuid.NewString(), s.now().UTC())
	if err := s.checks.Save(ctx, check); err != nil {
		return CheckResult{}, err
	}
	result.CheckID = check.CheckID

	if s.audit != nil {
		if err := s.audit.Log(ctx, audit.Input{
			ActorUserID:     result.BuyerUserID,
			ResourceType:    "buyer_check",
			ResourceID:      result.CheckID,
			ResourceVersion: 1,
			Action:          "buyer_check.completed",
			Status:          "completed",
			Payload:         audit.MutationPayload{After: check},
		}); err != nil {
			return CheckResult{}, err
		}
	}

	return result, nil
}

func (r CheckResult) toBuyerCheck(checkID string, createdAt time.Time) domain.BuyerCheck {
	return domain.BuyerCheck{
		CheckID:          checkID,
		ShopID:           r.ShopID,
		ProductID:        r.ProductID,
		PledgeID:         r.PledgeID,
		BuyerUserID:      r.BuyerUserID,
		Status:           BuyerCheckStatusCompleted,
		Version:          1,
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
		ImageHash:        r.ImageHash,
		ImageCID:         r.ImageCID,
		Reasons:          r.Reasons,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
}

func (s *Service) ensureQuota(ctx context.Context, buyerUserID string) error {
	checks, err := s.checks.ListByBuyerUserID(ctx, buyerUserID)
	if err != nil {
		return err
	}
	since := s.now().UTC().Add(-1 * time.Hour)
	count := 0
	for _, check := range checks {
		if check.CreatedAt.After(since) {
			count++
		}
	}
	if count >= 10 {
		return ErrRateLimited
	}
	return nil
}

func (s *Service) ensureAdmin(ctx context.Context, userID string) error {
	user, err := s.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func validateModerationStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case BuyerCheckStatusCompleted, BuyerCheckStatusFlagged, BuyerCheckStatusRejected:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: invalid status", ErrInvalidCheck)
	}
}
