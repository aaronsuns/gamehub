package config

import (
	"os"
	"time"
)

// Debug returns true when GAMEHUB_DEBUG is set (e.g. GAMEHUB_DEBUG=1).
func Debug() bool {
	return os.Getenv("GAMEHUB_DEBUG") != ""
}

const (
	// Pagination: Atlas API max page size [0, 50].
	PageSize = 50

	// Inbound rate limit per IP.
	InboundRateLimitRequests = 60
	InboundRateLimitPer      = time.Minute
	InboundRetryAfterSec     = 60

	// Live context cache TTL.
	LiveCacheTTL = 10 * time.Second

	// Atlas API client HTTP timeout.
	AtlasClientTimeout = 30 * time.Second
)
