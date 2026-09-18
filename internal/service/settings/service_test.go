package settings

import (
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
	"vngrocery/internal/service/audit"
)

type repoStub struct {
	current domain.RuntimeSettings
	saved   *domain.RuntimeSettings
}

func (r *repoStub) Get(ctx context.Context) (domain.RuntimeSettings, error) {
	return r.current, nil
}

func (r *repoStub) Save(ctx context.Context, settings domain.RuntimeSettings) error {
	r.saved = &settings
	return nil
}

type auditStub struct{ inputs []audit.Input }

func (a *auditStub) Log(ctx context.Context, input audit.Input) error {
	a.inputs = append(a.inputs, input)
	return nil
}

// Changing a rate limit is an operational decision, so the record has to show
// who made it and what it was before.
func TestUpdateRecordsBeforeAndAfter(t *testing.T) {
	repo := &repoStub{current: domain.DefaultRuntimeSettings()}
	auditor := &auditStub{}

	after, err := NewService(repo, auditor).Update(context.Background(), UpdateInput{
		ActorUserID:            "admin-1",
		FreshnessReportPerHour: 30,
		BuyerCheckPerHour:      40,
		RateLimitWindowMinutes: 15,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if after.FreshnessReportPerHour != 30 || after.BuyerCheckPerHour != 40 || after.RateLimitWindowMinutes != 15 {
		t.Fatalf("saved values did not survive the round trip: %#v", after)
	}
	if repo.saved == nil || repo.saved.UpdatedByUserID != "admin-1" {
		t.Fatalf("expected the actor to be stored, got %#v", repo.saved)
	}
	if len(auditor.inputs) != 1 {
		t.Fatalf("expected exactly one audit event, got %d", len(auditor.inputs))
	}
	payload, ok := auditor.inputs[0].Payload.(audit.MutationPayload)
	if !ok {
		t.Fatalf("expected a mutation payload, got %T", auditor.inputs[0].Payload)
	}
	before, ok := payload.Before.(domain.RuntimeSettings)
	if !ok || before.BuyerCheckPerHour != domain.DefaultBuyerCheckPerHour {
		t.Fatalf("expected the previous limits in the event, got %#v", payload.Before)
	}
}

// A zero would read as "no requests allowed" and lock every buyer out, so it is
// rejected at the edge rather than normalised away in silence.
func TestUpdateRejectsNonPositiveLimits(t *testing.T) {
	repo := &repoStub{current: domain.DefaultRuntimeSettings()}
	_, err := NewService(repo, &auditStub{}).Update(context.Background(), UpdateInput{
		ActorUserID:            "admin-1",
		FreshnessReportPerHour: 0,
		BuyerCheckPerHour:      10,
		RateLimitWindowMinutes: 60,
	})
	if !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected an invalid settings error, got %v", err)
	}
	if repo.saved != nil {
		t.Fatalf("a rejected update must not be written: %#v", repo.saved)
	}
}
