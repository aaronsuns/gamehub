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

// SeriesLive returns currently live/ongoing series (from cache when valid).
func (h *Handler) SeriesLive(c *gin.Context) {
	body, err := h.Live.GetLiveSeries(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

// PlayersLive returns players currently playing in live series (from cache when valid).
func (h *Handler) PlayersLive(c *gin.Context) {
	body, err := h.Live.GetLivePlayers(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

// TeamsLive returns teams currently playing in live series (from cache when valid).
func (h *Handler) TeamsLive(c *gin.Context) {
	body, err := h.Live.GetLiveTeams(c.Request.Context())
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
