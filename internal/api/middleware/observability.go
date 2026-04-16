package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	gofirestore "cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Metrics struct {
	requestsTotal            uint64
	errorsTotal              uint64
	rateLimitRejectionsTotal uint64
	bundleTokenIssuedTotal   uint64
	bundleTokenReplayTotal   uint64
	buyerCheckRetriedTotal   uint64
	integrityAnchorAttempts  uint64
	integrityAnchorSuccess   uint64
	integrityAnchorFailures  uint64
	integrityVerifyMismatch  uint64
	integrityReanchorTotal   uint64
	integrityRevokeTotal     uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK,
			"requests_total %d\nerrors_total %d\nrate_limit_rejections_total %d\nbundle_token_issued_total %d\nbundle_token_replay_total %d\nbuyer_check_retried_total %d\nintegrity_anchor_attempts_total %d\nintegrity_anchor_success_total %d\nintegrity_anchor_failures_total %d\nintegrity_verify_mismatch_total %d\nintegrity_reanchor_total %d\nintegrity_revoke_total %d\n",
			atomic.LoadUint64(&m.requestsTotal),
			atomic.LoadUint64(&m.errorsTotal),
			atomic.LoadUint64(&m.rateLimitRejectionsTotal),
			atomic.LoadUint64(&m.bundleTokenIssuedTotal),
			atomic.LoadUint64(&m.bundleTokenReplayTotal),
			atomic.LoadUint64(&m.buyerCheckRetriedTotal),
			atomic.LoadUint64(&m.integrityAnchorAttempts),
			atomic.LoadUint64(&m.integrityAnchorSuccess),
			atomic.LoadUint64(&m.integrityAnchorFailures),
			atomic.LoadUint64(&m.integrityVerifyMismatch),
			atomic.LoadUint64(&m.integrityReanchorTotal),
			atomic.LoadUint64(&m.integrityRevokeTotal),
		)
	}
}

func (m *Metrics) IncBundleTokenIssued() {
	if m != nil {
		atomic.AddUint64(&m.bundleTokenIssuedTotal, 1)
	}
}

func (m *Metrics) IncBundleTokenReplay() {
	if m != nil {
		atomic.AddUint64(&m.bundleTokenReplayTotal, 1)
	}
}

func (m *Metrics) IncBuyerCheckRetried() {
	if m != nil {
		atomic.AddUint64(&m.buyerCheckRetriedTotal, 1)
	}
}

func StructuredLogger(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-Id", requestID)
		c.Next()
		status := c.Writer.Status()
		if metrics != nil {
			atomic.AddUint64(&metrics.requestsTotal, 1)
			if status >= http.StatusBadRequest {
				atomic.AddUint64(&metrics.errorsTotal, 1)
			}
		}
		log.Printf(`{"request_id":%q,"method":%q,"path":%q,"status":%d,"latency_ms":%d,"client_ip":%q}`, requestID, c.Request.Method, c.FullPath(), status, time.Since(start).Milliseconds(), c.ClientIP())
	}
}

func (m *Metrics) IncRateLimitRejection() {
	if m != nil {
		atomic.AddUint64(&m.rateLimitRejectionsTotal, 1)
	}
}

func (m *Metrics) IncIntegrityAnchorAttempt() {
	if m != nil {
		atomic.AddUint64(&m.integrityAnchorAttempts, 1)
	}
}

func (m *Metrics) IncIntegrityAnchorSuccess() {
	if m != nil {
		atomic.AddUint64(&m.integrityAnchorSuccess, 1)
	}
}

func (m *Metrics) IncIntegrityAnchorFailure() {
	if m != nil {
		atomic.AddUint64(&m.integrityAnchorFailures, 1)
	}
}

func (m *Metrics) IncIntegrityVerifyMismatch() {
	if m != nil {
		atomic.AddUint64(&m.integrityVerifyMismatch, 1)
	}
}

func (m *Metrics) IncIntegrityReanchor() {
	if m != nil {
		atomic.AddUint64(&m.integrityReanchorTotal, 1)
	}
}

func (m *Metrics) IncIntegrityRevoke() {
	if m != nil {
		atomic.AddUint64(&m.integrityRevokeTotal, 1)
	}
}

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

type RateLimitStore interface {
	Allow(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, int, error)
}

type memoryRateLimitStore struct {
	mu      sync.Mutex
	clients map[string]rateLimitEntry
}

func NewMemoryRateLimitStore() RateLimitStore {
	return &memoryRateLimitStore{clients: map[string]rateLimitEntry{}}
}

func (s *memoryRateLimitStore) Allow(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, int, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	entry := s.clients[key]
	if now.Sub(entry.windowStart) >= window {
		entry = rateLimitEntry{windowStart: now}
	}
	entry.count++
	s.clients[key] = entry
	return entry.count <= maxRequests, int(window.Seconds()), nil
}

type FirestoreRateLimitStore struct {
	client     *gofirestore.Client
	collection string
}

func NewFirestoreRateLimitStore(client *gofirestore.Client, collection string) *FirestoreRateLimitStore {
	return &FirestoreRateLimitStore{client: client, collection: collection}
}

func (s *FirestoreRateLimitStore) Allow(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, int, error) {
	if s == nil || s.client == nil {
		return true, int(window.Seconds()), nil
	}
	bucket := time.Now().UTC().Truncate(window)
	docID := key + ":" + bucket.Format(time.RFC3339)
	docRef := s.client.Collection(s.collection).Doc(docID)
	var count int
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *gofirestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			count = 1
			return tx.Set(docRef, map[string]any{
				"key":         key,
				"bucketStart": bucket,
				"count":       count,
				"expireAt":    bucket.Add(window),
			})
		}
		current, _ := snap.DataAt("count")
		if v, ok := current.(int64); ok {
			count = int(v) + 1
		} else {
			count = 1
		}
		return tx.Set(docRef, map[string]any{
			"key":         key,
			"bucketStart": bucket,
			"count":       count,
			"expireAt":    bucket.Add(window),
		}, gofirestore.MergeAll)
	})
	if err != nil {
		return false, int(window.Seconds()), err
	}
	return count <= maxRequests, int(window.Seconds()), nil
}

func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return RateLimitWithStore(NewMemoryRateLimitStore(), maxRequests, window, nil)
}

func RateLimitWithStore(store RateLimitStore, maxRequests int, window time.Duration, metrics *Metrics) gin.HandlerFunc {
	if store == nil {
		store = NewMemoryRateLimitStore()
	}

	return func(c *gin.Context) {
		if maxRequests <= 0 || window <= 0 {
			c.Next()
			return
		}

		key := c.ClientIP()
		if principal, ok := GetPrincipal(c); ok && principal.UserID != "" {
			key = "user:" + principal.UserID
		}
		allowed, retryAfter, err := store.Allow(c.Request.Context(), key, maxRequests, window)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limit backend unavailable"})
			return
		}

		if !allowed {
			if metrics != nil {
				metrics.IncRateLimitRejection()
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
