package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aaron/gamehub/internal/config"
	"github.com/aaron/gamehub/internal/metrics"
)

// Limiter implements a per-IP rate limit using a fixed window: at most
// requests in any contiguous period of length per.
type Limiter struct {
	requests int
	per      time.Duration
	mu       sync.Mutex
	buckets  map[string]*bucket
}

type bucket struct {
	count      int
	windowStart time.Time
}

// NewLimiter creates a rate limiter allowing at most requests per IP per window.
// Example: NewLimiter(120, time.Minute) = at most 120 requests in any 1-minute window per IP.
func NewLimiter(requests int, per time.Duration) *Limiter {
	return &Limiter{
		requests: requests,
		per:      per,
		buckets:  make(map[string]*bucket),
	}
}

// Allow reports whether the request from ip should be allowed.
// Returns true if allowed, false if rate limited.
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Opportunistic eviction of stale buckets to prevent unbounded memory growth.
	if len(l.buckets) > config.InboundBucketEvictThreshold() {
		l.evictStaleLocked()
	}

	now := time.Now()
	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &bucket{count: 1, windowStart: now}
		return true
	}

	// If we're past the current window, start a new one.
	if now.Sub(b.windowStart) >= l.per {
		b.count = 0
		b.windowStart = now
	}

	b.count++
	if b.count > l.requests {
		b.count-- // don't count this request
		return false
	}
	return true
}

// bucketCount returns the number of buckets (for testing).
func (l *Limiter) bucketCount() int {
	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	return n
}

// evictStaleLocked removes buckets whose window start is older than InboundBucketMaxStale.
func (l *Limiter) evictStaleLocked() {
	cutoff := time.Now().Add(-config.InboundBucketMaxStale())
	for ip, b := range l.buckets {
		if b.windowStart.Before(cutoff) {
			delete(l.buckets, ip)
		}
	}
}

// Middleware returns a Gin middleware that rate limits by client IP.
func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c.Request)
		if !l.Allow(ip) {
			metrics.Inbound429.Add(1)
			retrySec := config.InboundRetryAfterSec()
			metrics.RecordInboundRetryAfter(retrySec)
			c.Header("Retry-After", fmt.Sprintf("%d", retrySec))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
