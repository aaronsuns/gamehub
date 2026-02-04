package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aaron/gamehub/internal/atlas"
	"github.com/aaron/gamehub/internal/live"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Atlas *atlas.Client
	Live  *live.Service
}

// New creates a new Handler.
func New(atlasClient *atlas.Client, liveService *live.Service) *Handler {
	return &Handler{Atlas: atlasClient, Live: liveService}
}

// SeriesLive returns currently live/ongoing series.
func (h *Handler) SeriesLive(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{"filter": "lifecycle=live"}
	body, rl, err := h.Atlas.GetSeriesAll(r.Context(), params)
	if err != nil {
		writeError(w, err, rl)
		return
	}
	writeJSON(w, body, rl)
}

// PlayersLive returns players currently playing in live series.
func (h *Handler) PlayersLive(w http.ResponseWriter, r *http.Request) {
	body, rl, err := h.fetchLiveEntities(r, "players")
	if err != nil {
		writeError(w, err, rl)
		return
	}
	writeJSON(w, body, rl)
}

// TeamsLive returns teams currently playing in live series.
func (h *Handler) TeamsLive(w http.ResponseWriter, r *http.Request) {
	body, rl, err := h.fetchLiveEntities(r, "teams")
	if err != nil {
		writeError(w, err, rl)
		return
	}
	writeJSON(w, body, rl)
}

func (h *Handler) fetchLiveEntities(r *http.Request, kind string) ([]byte, *atlas.RateLimit, error) {
	ctx := r.Context()
	liveCtx, err := h.Live.GetLiveContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	var params map[string]string
	var ids []int
	switch kind {
	case "players":
		ids = liveCtx.PlayerIDs
		params = make(map[string]string)
		if len(ids) > 0 {
			params["filter"] = atlas.FilterIDIn(ids)
		}
	case "teams":
		ids = liveCtx.TeamIDs
		params = make(map[string]string)
		if len(ids) > 0 {
			params["filter"] = atlas.FilterIDIn(ids)
		}
	default:
		return nil, nil, fmt.Errorf("unknown kind: %s", kind)
	}
	if len(ids) == 0 {
		empty := []interface{}{}
		body, _ := json.Marshal(empty)
		return body, nil, nil
	}
	var body []byte
	var rl *atlas.RateLimit
	if kind == "players" {
		body, rl, err = h.Atlas.GetPlayersAll(ctx, params)
	} else {
		body, rl, err = h.Atlas.GetTeamsAll(ctx, params)
	}
	if err != nil {
		return nil, rl, err
	}
	return body, rl, nil
}

func writeJSON(w http.ResponseWriter, body []byte, rl *atlas.RateLimit) {
	if rl != nil {
		if rl.Limit > 0 {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.Limit))
		}
		if rl.Remaining >= 0 {
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rl.Remaining))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error, rl *atlas.RateLimit) {
	if rlErr, ok := err.(*atlas.ErrRateLimited); ok {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", rlErr.RetryAfterMs))
		if rl != nil {
			w.Header().Set("X-RateLimit-Remaining", "0")
		}
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
