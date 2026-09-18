package domain

import "time"

// RuntimeSettingsID is the only document in the settings collection.
//
// A single fixed row rather than a general key/value config store: the values
// below are the ones an operator has a reason to change while the stack is
// running, and a typed struct keeps them checkable at compile time.
const RuntimeSettingsID = "runtime"

// Defaults match the constants these settings replaced, so turning the feature
// on changes no behaviour until someone deliberately edits a value.
const (
	DefaultFreshnessReportPerHour = 10
	DefaultBuyerCheckPerHour      = 10
	DefaultRateLimitWindowMinutes = 60
)

// RuntimeSettings holds operational limits that can be changed without a
// redeploy.
//
// Every change is written to the signed event log like any other mutation:
// raising a rate limit is an operational decision with consequences, so who
// changed it, when, and from what has to be traceable.
type RuntimeSettings struct {
	ID string `firestore:"id"`

	// Per user, per window. Zero or negative means "fall back to the default"
	// rather than "no requests allowed" - a mistyped 0 should not lock every
	// buyer out of the feature.
	FreshnessReportPerHour int `firestore:"freshnessReportPerHour"`
	BuyerCheckPerHour      int `firestore:"buyerCheckPerHour"`

	// The window both limits are counted over.
	RateLimitWindowMinutes int `firestore:"rateLimitWindowMinutes"`

	// Counts from 1 on the first save. The event log refuses a resourceVersion
	// of 0, and without it a settings change was being written unsigned.
	Version int `firestore:"version"`

	UpdatedByUserID string    `firestore:"updatedByUserId"`
	UpdatedAt       time.Time `firestore:"updatedAt"`
}

// DefaultRuntimeSettings is what the system runs on before an admin has saved
// anything, and what a field falls back to when it holds a value that cannot
// be honoured.
func DefaultRuntimeSettings() RuntimeSettings {
	return RuntimeSettings{
		ID:                     RuntimeSettingsID,
		FreshnessReportPerHour: DefaultFreshnessReportPerHour,
		BuyerCheckPerHour:      DefaultBuyerCheckPerHour,
		RateLimitWindowMinutes: DefaultRateLimitWindowMinutes,
	}
}

// Normalized returns a copy with unusable values replaced by their defaults.
//
// Applied on read rather than only on write: a document saved by an older
// build, or edited straight in the database, must not be able to take the
// feature down.
func (s RuntimeSettings) Normalized() RuntimeSettings {
	defaults := DefaultRuntimeSettings()
	s.ID = RuntimeSettingsID
	if s.FreshnessReportPerHour <= 0 {
		s.FreshnessReportPerHour = defaults.FreshnessReportPerHour
	}
	if s.BuyerCheckPerHour <= 0 {
		s.BuyerCheckPerHour = defaults.BuyerCheckPerHour
	}
	if s.RateLimitWindowMinutes <= 0 {
		s.RateLimitWindowMinutes = defaults.RateLimitWindowMinutes
	}
	return s
}

// RateLimitWindow is the window as a duration, ready to subtract from now.
func (s RuntimeSettings) RateLimitWindow() time.Duration {
	return time.Duration(s.Normalized().RateLimitWindowMinutes) * time.Minute
}
