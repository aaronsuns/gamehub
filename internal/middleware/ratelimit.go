package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aaron/gamehub/internal/config"
	"github.com/aaron/gamehub/internal/metrics"
)

// Limiter implements a static per-IP rate limit using a token bucket.
type Limiter struct {
	requests int
	per      time.Duration
	mu       sync.Mutex
	buckets  map[string]*bucket
}

type bucket struct {
	tokens   int
	lastFill time.Time
}

// NewLimiter creates a rate limiter allowing requests per IP per window.
// Example: NewLimiter(60, time.Minute) = 60 req/min per IP.
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

	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &bucket{tokens: l.requests - 1, lastFill: time.Now()}
		return true
	}

	// Refill tokens based on elapsed time
	// Interval per token = l.per / l.requests
	elapsed := time.Since(b.lastFill)
	interval := l.per.Nanoseconds() / int64(l.requests)
	if interval <= 0 {
		interval = 1
	}
	refill := int(elapsed.Nanoseconds() / interval)
	if refill > 0 {
		b.tokens += refill
		if b.tokens > l.requests {
			b.tokens = l.requests
		}
		b.lastFill = time.Now()
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// bucketCount returns the number of buckets (for testing).
func (l *Limiter) bucketCount() int {
	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	return n
}

// evictStaleLocked removes buckets unused for InboundBucketMaxStale.
func (l *Limiter) evictStaleLocked() {
	cutoff := time.Now().Add(-config.InboundBucketMaxStale())
	for ip, b := range l.buckets {
		if b.lastFill.Before(cutoff) {
			delete(l.buckets, ip)
		}
	}
}

// Middleware returns an HTTP middleware that rate limits by client IP.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !l.Allow(ip) {
			metrics.Inbound429.Add(1)
			retrySec := config.InboundRetryAfterSec()
			metrics.RecordInboundRetryAfter(retrySec)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySec))
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
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
