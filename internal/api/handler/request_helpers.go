package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/domain"
)

// writeRateLimited answers 429 with the wait, and reports whether it handled
// the error.
//
// The wait is sent two ways on purpose: Retry-After is the standard header any
// client or proxy already understands, and retryAfterMinutes is what the app
// puts in front of the person holding the phone, who needs a number in their
// own language rather than "too many requests".
func writeRateLimited(c *gin.Context, err error) bool {
	if !errors.Is(err, domain.ErrRateLimited) {
		return false
	}
	var limited domain.RateLimitedError
	if !errors.As(err, &limited) {
		// Matched the sentinel without carrying a wait; the window is the most
		// honest fallback we have.
		limited = domain.RateLimitedError{RetryAfter: domain.DefaultRuntimeSettings().RateLimitWindow()}
	}
	minutes := limited.RetryAfterMinutes()
	c.Header("Retry-After", strconv.Itoa(minutes*60))
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":             err.Error(),
		"retryAfterMinutes": minutes,
	})
	return true
}

func parsePositiveIntQuery(raw, field string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", field)
	}
	return parsed, nil
}

// parseOptionalPositiveIntQuery reads a query parameter that may be absent.
//
// Absent means "use the default", which is not the same as a caller sending
// nonsense: an unparseable value is still an error rather than being quietly
// replaced by the default.
func parseOptionalPositiveIntQuery(raw, field string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return parsePositiveIntQuery(raw, field)
}
