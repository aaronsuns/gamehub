package handlers

import (
	"fmt"
	"log"
	"net/http"

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
	body, _, err := h.Atlas.GetSeriesAll(r.Context(), params)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, body)
}

// PlayersLive returns players currently playing in live series.
func (h *Handler) PlayersLive(w http.ResponseWriter, r *http.Request) {
	liveCtx, err := h.Live.GetLiveContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if len(liveCtx.PlayerIDs) == 0 {
		writeJSON(w, []byte("[]"))
		return
	}
	params := map[string]string{"filter": atlas.FilterIDIn(liveCtx.PlayerIDs)}
	body, _, err := h.Atlas.GetPlayersAll(r.Context(), params)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, body)
}

// TeamsLive returns teams currently playing in live series.
func (h *Handler) TeamsLive(w http.ResponseWriter, r *http.Request) {
	liveCtx, err := h.Live.GetLiveContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if len(liveCtx.TeamIDs) == 0 {
		writeJSON(w, []byte("[]"))
		return
	}
	params := map[string]string{"filter": atlas.FilterIDIn(liveCtx.TeamIDs)}
	body, _, err := h.Atlas.GetTeamsAll(r.Context(), params)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, body)
}

func writeJSON(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if rlErr, ok := err.(*atlas.ErrRateLimited); ok {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", rlErr.RetryAfterMs))
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
