package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	limiter := NewLimiter(3, time.Second) // 3 requests per second

	ip := "192.168.1.1"
	if !limiter.Allow(ip) {
		t.Error("first request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("second request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("third request should be allowed")
	}
	if limiter.Allow(ip) {
		t.Error("fourth request should be denied")
	}
}

func TestLimiter_DifferentIPs(t *testing.T) {
	limiter := NewLimiter(2, time.Second)

	ip1 := "10.0.0.1"
	ip2 := "10.0.0.2"

	if !limiter.Allow(ip1) {
		t.Error("ip1 first should be allowed")
	}
	if !limiter.Allow(ip1) {
		t.Error("ip1 second should be allowed")
	}
	if limiter.Allow(ip1) {
		t.Error("ip1 third should be denied")
	}

	if !limiter.Allow(ip2) {
		t.Error("ip2 first should be allowed (different IP)")
	}
	if !limiter.Allow(ip2) {
		t.Error("ip2 second should be allowed")
	}
	if limiter.Allow(ip2) {
		t.Error("ip2 third should be denied")
	}
}

func TestMiddleware_Returns429(t *testing.T) {
	limiter := NewLimiter(2, time.Second)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// First two should succeed
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: want 200, got %d", i+1, rec.Code)
		}
	}

	// Third should be 429
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("want 429, got %d", rec.Code)
	}
	if retry := rec.Header().Get("Retry-After"); retry != "60" {
		t.Errorf("want Retry-After: 60, got %q", retry)
	}
}
