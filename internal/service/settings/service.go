// Package settings holds the operational limits an admin can change without a
// redeploy.
package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

var ErrInvalidSettings = errors.New("invalid settings")

// Auditor is the part of the audit service this package needs.
type Auditor interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	repo  repository.SettingsRepository
	audit Auditor
	now   func() time.Time
}

func NewService(repo repository.SettingsRepository, auditor Auditor) *Service {
	return &Service{repo: repo, audit: auditor, now: time.Now}
}

func (s *Service) Get(ctx context.Context) (domain.RuntimeSettings, error) {
	return s.repo.Get(ctx)
}

type UpdateInput struct {
	ActorUserID            string
	FreshnessReportPerHour int
	BuyerCheckPerHour      int
	RateLimitWindowMinutes int
}

// Update replaces the settings document and records who changed what.
//
// Raising a rate limit is an operational decision with consequences for every
// record created afterwards, so it goes through the same signed event log as a
// product edit rather than being a silent config tweak.
func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.RuntimeSettings, error) {
	actor := strings.TrimSpace(input.ActorUserID)
	if actor == "" {
		return domain.RuntimeSettings{}, fmt.Errorf("%w: actorUserId is required", ErrInvalidSettings)
	}
	for label, value := range map[string]int{
		"freshnessReportPerHour": input.FreshnessReportPerHour,
		"buyerCheckPerHour":      input.BuyerCheckPerHour,
		"rateLimitWindowMinutes": input.RateLimitWindowMinutes,
	} {
		if value <= 0 {
			return domain.RuntimeSettings{}, fmt.Errorf("%w: %s must be greater than 0", ErrInvalidSettings, label)
		}
	}

	before, err := s.repo.Get(ctx)
	if err != nil {
		return domain.RuntimeSettings{}, err
	}

	after := domain.RuntimeSettings{
		ID:                     domain.RuntimeSettingsID,
		FreshnessReportPerHour: input.FreshnessReportPerHour,
		BuyerCheckPerHour:      input.BuyerCheckPerHour,
		RateLimitWindowMinutes: input.RateLimitWindowMinutes,
		Version:                before.Version + 1,
		UpdatedByUserID:        actor,
		UpdatedAt:              s.now().UTC(),
	}

	// Signed before it is stored, not after: a save that succeeded while the
	// event failed would leave a limit in force that the record cannot explain.
	if s.audit != nil {
		if err := s.audit.Log(ctx, audit.Input{
			ActorUserID:     actor,
			ResourceType:    "runtime_settings",
			ResourceID:      domain.RuntimeSettingsID,
			ResourceVersion: after.Version,
			Action:          "settings.update",
			Status:          "succeeded",
			Payload:         audit.MutationPayload{Before: before, After: after},
		}); err != nil {
			return domain.RuntimeSettings{}, err
		}
	}
	if err := s.repo.Save(ctx, after); err != nil {
		return domain.RuntimeSettings{}, err
	}
	return after, nil
}
