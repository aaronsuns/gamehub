package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaron/gamehub/internal/atlas"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Errorf("body: want %q, got %q", `{"status":"ok"}`, body)
	}
}

func TestWriteError_RateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	err := &atlas.ErrRateLimited{RetryAfterMs: 500}
	writeError(w, err)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("want 429, got %d", w.Code)
	}
	if retry := w.Header().Get("Retry-After"); retry != "500" {
		t.Errorf("want Retry-After: 500, got %q", retry)
	}
}
