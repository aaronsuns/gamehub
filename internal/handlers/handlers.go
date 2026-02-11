package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

// Health returns 200 OK for liveness/readiness probes.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SeriesLive returns currently live/ongoing series.
func (h *Handler) SeriesLive(c *gin.Context) {
	params := map[string]string{"filter": "lifecycle=live"}
	body, _, err := h.Atlas.GetSeriesAll(c.Request.Context(), params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

// PlayersLive returns players currently playing in live series.
func (h *Handler) PlayersLive(c *gin.Context) {
	liveCtx, err := h.Live.GetLiveContext(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	if len(liveCtx.PlayerIDs) == 0 {
		c.Data(http.StatusOK, "application/json", []byte("[]"))
		return
	}
	params := map[string]string{"filter": atlas.FilterIDIn(liveCtx.PlayerIDs)}
	body, _, err := h.Atlas.GetPlayersAll(c.Request.Context(), params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

// TeamsLive returns teams currently playing in live series.
func (h *Handler) TeamsLive(c *gin.Context) {
	liveCtx, err := h.Live.GetLiveContext(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	if len(liveCtx.TeamIDs) == 0 {
		c.Data(http.StatusOK, "application/json", []byte("[]"))
		return
	}
	params := map[string]string{"filter": atlas.FilterIDIn(liveCtx.TeamIDs)}
	body, _, err := h.Atlas.GetTeamsAll(c.Request.Context(), params)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

func writeError(c *gin.Context, err error) {
	if rlErr, ok := err.(*atlas.ErrRateLimited); ok {
		c.Header("Retry-After", fmt.Sprintf("%d", rlErr.RetryAfterMs))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
