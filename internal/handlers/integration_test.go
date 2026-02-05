package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaron/gamehub/internal/atlas"
	"github.com/aaron/gamehub/internal/config"
	"github.com/aaron/gamehub/internal/live"
)

func itemName(path string, m map[string]interface{}) string {
	switch {
	case strings.Contains(path, "series"):
		if s, ok := m["title"].(string); ok && s != "" {
			return s
		}
	case strings.Contains(path, "players"):
		if s, ok := m["nick_name"].(string); ok && s != "" {
			return s
		}
		if f, _ := m["first_name"].(string); f != "" {
			if l, _ := m["last_name"].(string); l != "" {
				return f + " " + l
			}
			return f
		}
	case strings.Contains(path, "teams"):
		if s, ok := m["name"].(string); ok && s != "" {
			return s
		}
	}
	if id, ok := m["id"].(float64); ok {
		return fmt.Sprintf("id:%.0f", id)
	}
	return "?"
}

// Integration tests against the real Atlas API.
// Skipped when ATLAS_API_KEY is not set.
// Writes responses to internal/handlers/integration/ for inspection.

func TestIntegration_LiveEndpoints(t *testing.T) {
	secret := os.Getenv("ATLAS_API_KEY")
	if secret == "" {
		t.Skip("ATLAS_API_KEY not set, skipping integration test")
	}

	client := atlas.NewClient(secret)
	liveSvc := live.NewService(client, config.LiveCacheTTL())
	h := New(client, liveSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /series/live", h.SeriesLive)
	mux.HandleFunc("GET /players/live", h.PlayersLive)
	mux.HandleFunc("GET /teams/live", h.TeamsLive)

	outDir := filepath.Join("integration")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	tests := []struct {
		name string
		path string
		file string
	}{
		{"series_live", "/series/live", "series_live.json"},
		{"players_live", "/players/live", "players_live.json"},
		{"teams_live", "/teams/live", "teams_live.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: want 200, got %d\nbody: %s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type: want application/json, got %s", ct)
			}

			var arr []json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &arr); err != nil {
				t.Fatalf("invalid JSON array: %v\nbody: %s", err, rec.Body.String())
			}

			if len(arr) == 0 {
				t.Errorf("%s: got 0 items (empty response is suspicious)", tt.path)
			}
			t.Logf("%s: %d items", tt.path, len(arr))

			var names []string
			const maxShow = 8
			for i, raw := range arr {
				var m map[string]interface{}
				if err := json.Unmarshal(raw, &m); err != nil {
					continue
				}
				name := itemName(tt.path, m)
				if i < maxShow {
					names = append(names, name)
				}
			}
			if len(names) > 0 {
				summary := strings.Join(names, ", ")
				if len(arr) > maxShow {
					summary += fmt.Sprintf(" ... (+%d more)", len(arr)-maxShow)
				}
				t.Logf("items: %s", summary)
			}

			body := rec.Body.Bytes()

			outPath := filepath.Join(outDir, tt.file)
			if err := os.WriteFile(outPath, body, 0644); err != nil {
				t.Logf("write result: %v", err)
			} else {
				t.Logf("result written to %s", outPath)
			}
		})
	}
}
