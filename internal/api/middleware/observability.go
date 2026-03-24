package middleware

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requestsTotal uint64
	errorsTotal   uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "requests_total %d\nerrors_total %d\n", atomic.LoadUint64(&m.requestsTotal), atomic.LoadUint64(&m.errorsTotal))
	}
}

func StructuredLogger(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		if metrics != nil {
			atomic.AddUint64(&metrics.requestsTotal, 1)
			if status >= http.StatusBadRequest {
				atomic.AddUint64(&metrics.errorsTotal, 1)
			}
		}
		log.Printf(`{"method":%q,"path":%q,"status":%d,"latency_ms":%d,"client_ip":%q}`, c.Request.Method, c.FullPath(), status, time.Since(start).Milliseconds(), c.ClientIP())
	}
}

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := map[string]rateLimitEntry{}

	return func(c *gin.Context) {
		if maxRequests <= 0 || window <= 0 {
			c.Next()
			return
		}

		now := time.Now()
		clientIP := c.ClientIP()
		mu.Lock()
		entry := clients[clientIP]
		if now.Sub(entry.windowStart) >= window {
			entry = rateLimitEntry{windowStart: now}
		}
		entry.count++
		clients[clientIP] = entry
		limited := entry.count > maxRequests
		retryAfter := int(window.Seconds())
		mu.Unlock()

		if limited {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
