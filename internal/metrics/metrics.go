package metrics

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
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
	}
}

// ServeJSON writes stats as JSON.
func ServeJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Stats())
}

// ServeMonitor writes the monitoring HTML page.
func ServeMonitor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(monitorHTML)
}

// responseRecorder captures status for metrics.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	if code == http.StatusOK {
		RequestsOK.Add(1)
	}
	r.ResponseWriter.WriteHeader(code)
}

// paths excluded from main traffic metrics (monitoring endpoints)
var excludedPaths = map[string]bool{"/stats": true, "/monitor": true}

// Middleware wraps a handler to count total requests and OK responses.
// /stats and /monitor are excluded so the main graph reflects only API traffic.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if excludedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		RequestsTotal.Add(1)
		rec := &responseRecorder{ResponseWriter: w, status: 0}
		next.ServeHTTP(rec, r)
	})
}
