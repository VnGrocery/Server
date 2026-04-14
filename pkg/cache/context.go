package cache

import (
	"context"
	"strings"
)

type contextKey string

const bypassCacheKey contextKey = "bypass_cache"

func WithBypass(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithValue(ctx, bypassCacheKey, enabled)
}

func ShouldBypass(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, ok := ctx.Value(bypassCacheKey).(bool)
	return ok && value
}

func ParseRealtimeFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
