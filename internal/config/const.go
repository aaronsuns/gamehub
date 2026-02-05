package config

import (
	"os"
	"strconv"
	"time"
)

// Debug returns true when GAMEHUB_DEBUG is set (e.g. GAMEHUB_DEBUG=1).
func Debug() bool {
	return os.Getenv("GAMEHUB_DEBUG") != ""
}

// envInt returns env value as int, or default if unset/invalid.
func envInt(name string, defaultVal int) int {
	if s := os.Getenv(name); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

// envDuration returns env value as duration, or default if unset/invalid.
func envDuration(name string, defaultVal time.Duration) time.Duration {
	if s := os.Getenv(name); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}

// PageSize returns Atlas API page size [0, 50]. Env: GAMEHUB_PAGE_SIZE.
func PageSize() int {
	return envInt("GAMEHUB_PAGE_SIZE", 50)
}

// InboundRateLimitRequests returns requests per IP per window. Env: GAMEHUB_INBOUND_RATE_LIMIT.
func InboundRateLimitRequests() int {
	return envInt("GAMEHUB_INBOUND_RATE_LIMIT", 60)
}

// InboundRateLimitPer returns the rate limit window. Env: GAMEHUB_INBOUND_RATE_LIMIT_PER.
func InboundRateLimitPer() time.Duration {
	return envDuration("GAMEHUB_INBOUND_RATE_LIMIT_PER", time.Minute)
}

// InboundRetryAfterSec returns Retry-After header value on 429. Env: GAMEHUB_INBOUND_RETRY_AFTER.
func InboundRetryAfterSec() int {
	return envInt("GAMEHUB_INBOUND_RETRY_AFTER", 60)
}

// InboundBucketMaxStale returns eviction threshold for stale buckets. Env: GAMEHUB_INBOUND_BUCKET_MAX_STALE.
func InboundBucketMaxStale() time.Duration {
	return envDuration("GAMEHUB_INBOUND_BUCKET_MAX_STALE", 5*time.Minute)
}

// InboundBucketEvictThreshold returns bucket count above which eviction runs. Env: GAMEHUB_INBOUND_BUCKET_EVICT_THRESHOLD.
func InboundBucketEvictThreshold() int {
	return envInt("GAMEHUB_INBOUND_BUCKET_EVICT_THRESHOLD", 100)
}

// LiveCacheTTL returns live context cache TTL. Env: GAMEHUB_LIVE_CACHE_TTL.
func LiveCacheTTL() time.Duration {
	return envDuration("GAMEHUB_LIVE_CACHE_TTL", 10*time.Second)
}

// AtlasClientTimeout returns Atlas API client HTTP timeout. Env: GAMEHUB_ATLAS_CLIENT_TIMEOUT.
func AtlasClientTimeout() time.Duration {
	return envDuration("GAMEHUB_ATLAS_CLIENT_TIMEOUT", 30*time.Second)
}

// AtlasOutboundMinBackoff returns minimum backoff on 429 when Retry-After is missing. Env: GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF.
func AtlasOutboundMinBackoff() time.Duration {
	return envDuration("GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF", time.Second)
}
