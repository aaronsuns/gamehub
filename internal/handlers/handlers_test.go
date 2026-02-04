package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaron/gamehub/internal/atlas"
)

func TestWriteError_RateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	err := &atlas.ErrRateLimited{RetryAfterMs: 500}
	writeError(w, err, nil)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("want 429, got %d", w.Code)
	}
	if retry := w.Header().Get("Retry-After"); retry != "500" {
		t.Errorf("want Retry-After: 500, got %q", retry)
	}
}
