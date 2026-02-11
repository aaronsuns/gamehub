package metrics

import (
	_ "embed"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aaron/gamehub/internal/config"
)

//go:embed monitor.html
var monitorHTML []byte

var (
	RequestsTotal         atomic.Uint64
	RequestsOK            atomic.Uint64
	Inbound429            atomic.Uint64
	Atlas429              atomic.Uint64
	LastInboundRetryAfter atomic.Uint64 // seconds we sent on our 429
	LastAtlasRetryAfter   atomic.Uint64 // ms Atlas told us to wait
)

// RecordInboundRetryAfter records the Retry-After we sent (seconds).
func RecordInboundRetryAfter(sec int) {
	LastInboundRetryAfter.Store(uint64(sec))
}

// RecordAtlasRetryAfter records the Retry-After we received from Atlas (ms).
func RecordAtlasRetryAfter(ms int) {
	LastAtlasRetryAfter.Store(uint64(ms))
}

const historySize = 120 // 2 min at 1 sample/sec

type sample struct {
	T                  int64  `json:"t"`
	Requests           uint64 `json:"req"`
	OK                 uint64 `json:"ok"`
	Inbound429         uint64 `json:"inbound_429"`
	Atlas429           uint64 `json:"atlas_429"`
	AtlasRetryAfterMs  uint64 `json:"atlas_retry_after_ms"`
	InboundRetryAfterS uint64 `json:"inbound_retry_after_s"`
}

var (
	history     [historySize]sample
	historyIdx  int
	historyMu   sync.Mutex
	lastTotal   uint64
	lastOK      uint64
	lastInbound uint64
	lastAtlas   uint64
)

func init() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			recordSample()
		}
	}()
}

func recordSample() {
	total := RequestsTotal.Load()
	ok := RequestsOK.Load()
	inbound := Inbound429.Load()
	atlas := Atlas429.Load()

	historyMu.Lock()
	defer historyMu.Unlock()

	history[historyIdx] = sample{
		T:                  time.Now().Unix(),
		Requests:           total - lastTotal,
		OK:                 ok - lastOK,
		Inbound429:         inbound - lastInbound,
		Atlas429:           atlas - lastAtlas,
		AtlasRetryAfterMs:  LastAtlasRetryAfter.Load(),
		InboundRetryAfterS: LastInboundRetryAfter.Load(),
	}
	historyIdx = (historyIdx + 1) % historySize
	lastTotal, lastOK, lastInbound, lastAtlas = total, ok, inbound, atlas
}

// Stats returns current counters and recent history for graphing.
func Stats() map[string]interface{} {
	historyMu.Lock()
	samples := make([]sample, historySize)
	n := 0
	for i := 0; i < historySize; i++ {
		idx := (historyIdx + i) % historySize
		if history[idx].T != 0 {
			samples[n] = history[idx]
			n++
		}
	}
	historyMu.Unlock()
	samples = samples[:n]

	return map[string]interface{}{
		"total": map[string]interface{}{
			"requests":              RequestsTotal.Load(),
			"ok":                    RequestsOK.Load(),
			"inbound_429":           Inbound429.Load(),
			"atlas_429":             Atlas429.Load(),
			"inbound_retry_after_s": LastInboundRetryAfter.Load(),
			"atlas_retry_after_ms":  LastAtlasRetryAfter.Load(),
		},
		"history": samples,
		"config":  getConfig(),
	}
}

// getConfig returns important configuration values affecting rate limiting and traffic load.
func getConfig() map[string]interface{} {
	inboundLimit := config.InboundRateLimitRequests()
	inboundWindow := config.InboundRateLimitPer()
	inboundRetryAfter := config.InboundRetryAfterSec()
	bucketMaxStale := config.InboundBucketMaxStale()
	bucketEvictThreshold := config.InboundBucketEvictThreshold()
	cacheTTL := config.LiveCacheTTL()
	pageSize := config.PageSize()
	minBackoff := config.AtlasOutboundMinBackoff()
	stressConcurrency := config.StressConcurrency()
	stressDelay := config.StressDelay()

	return map[string]interface{}{
		"inbound_rate_limit":        inboundLimit,
		"inbound_rate_window":       inboundWindow.String(),
		"inbound_retry_after":       inboundRetryAfter,
		"inbound_bucket_max_stale":  bucketMaxStale.String(),
		"inbound_bucket_evict_threshold": bucketEvictThreshold,
		"live_cache_ttl":            cacheTTL.String(),
		"atlas_page_size":           pageSize,
		"atlas_min_backoff":         minBackoff.String(),
		"stress_concurrency":        stressConcurrency,
		"stress_delay":              stressDelay.String(),
	}
}

// ServeJSON writes stats as JSON.
func ServeJSON(c *gin.Context) {
	c.JSON(http.StatusOK, Stats())
}

// ServeMonitor writes the monitoring HTML page.
func ServeMonitor(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", monitorHTML)
}

// paths excluded from main traffic metrics (monitoring endpoints)
var excludedPaths = map[string]bool{"/stats": true, "/monitor": true}

// Middleware wraps a handler to count total requests and OK responses.
// /stats and /monitor are excluded so the main graph reflects only API traffic.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if excludedPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		RequestsTotal.Add(1)
		c.Next()
		// Check status after the handler runs
		if c.Writer.Status() == http.StatusOK {
			RequestsOK.Add(1)
		}
	}
}
