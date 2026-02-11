package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/aaron/gamehub/internal/atlas"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
	
	Health(c)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Errorf("body: want %q, got %q", `{"status":"ok"}`, body)
	}
}

func TestWriteError_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	
	err := &atlas.ErrRateLimited{RetryAfterMs: 500}
	writeError(c, err)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("want 429, got %d", rec.Code)
	}
	if retry := rec.Header().Get("Retry-After"); retry != "500" {
		t.Errorf("want Retry-After: 500, got %q", retry)
	}
}
