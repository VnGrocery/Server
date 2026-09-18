package domain

import (
	"errors"
	"time"
)

// ErrRateLimited is what every per-user quota in the system returns, so a
// handler can answer 429 without knowing which feature was throttled.
var ErrRateLimited = errors.New("rate limit exceeded")

// RateLimitedError carries how long the caller has to wait.
//
// "Too many requests" on its own leaves someone tapping the button again
// every few seconds. The wait is the only part of the answer they can act on,
// so it travels with the error rather than being guessed at the edge.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e RateLimitedError) Error() string { return "rate limit exceeded" }

// Is makes errors.Is(err, ErrRateLimited) true for this type, so existing
// handler branches keep matching.
func (e RateLimitedError) Is(target error) bool { return target == ErrRateLimited }

// RetryAfterMinutes is the wait rounded up to a whole minute, never below 1.
//
// Rounded up because rounding down tells someone to retry while still
// blocked; a floor of 1 because "retry in 0 minutes" reads as a broken screen.
func (e RateLimitedError) RetryAfterMinutes() int {
	minutes := int((e.RetryAfter + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		return 1
	}
	return minutes
}
