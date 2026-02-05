package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/aaron/gamehub/internal/config"
	"github.com/aaron/gamehub/internal/metrics"
)

const (
	defaultBaseURL = "https://atlas.abiosgaming.com/v3"
)

// RateLimit holds rate limit info from response headers.
type RateLimit struct {
	Limit     int
	Burst     int
	Remaining int
	ResetMs   int
}

// Client is an Atlas API client with reactive outbound rate limiting.
// Throttles only after receiving 429 from Atlas; respects Retry-After.
type Client struct {
	baseURL         string
	secret          string
	httpClient      *http.Client
	outMu           sync.Mutex
	outBackoffUntil time.Time // don't send before this (zero = no backoff)
}

// NewClient creates an Atlas API client.
func NewClient(secret string) *Client {
	return NewClientWithURL(secret, defaultBaseURL)
}

// NewClientWithURL creates a client with a custom base URL.
func NewClientWithURL(secret, baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: config.AtlasClientTimeout(),
		},
	}
}

// ErrRateLimited is returned when the API returns 429.
type ErrRateLimited struct {
	RetryAfterMs int
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited: retry after %d ms", e.RetryAfterMs)
}

// Get performs a GET request and returns body, rate limit info, and error.
// On 429, returns ErrRateLimited with RetryAfterMs from the Retry-After header.
func (c *Client) Get(ctx context.Context, path string) ([]byte, *RateLimit, error) {
	if err := c.waitOutbound(ctx); err != nil {
		return nil, nil, err
	}
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Abios-Secret", c.secret)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	rl := parseRateLimit(resp.Header)

	if resp.StatusCode == http.StatusTooManyRequests {
		retryMs := parseRetryAfter(resp.Header.Get("Retry-After"))
		c.setBackoff(retryMs)
		metrics.Atlas429.Add(1)
		metrics.RecordAtlasRetryAfter(retryMs)
		return nil, rl, &ErrRateLimited{RetryAfterMs: retryMs}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, rl, fmt.Errorf("atlas API error: status %d: %s", resp.StatusCode, string(body))
	}

	return body, rl, nil
}

// waitOutbound waits until any active backoff (from 429) has elapsed.
func (c *Client) waitOutbound(ctx context.Context) error {
	c.outMu.Lock()
	until := c.outBackoffUntil
	c.outMu.Unlock()
	if until.IsZero() || time.Now().After(until) {
		return nil
	}
	sleep := time.Until(until)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleep):
	}
	return nil
}

// setBackoff records that we received 429; next request will wait retryMs.
func (c *Client) setBackoff(retryMs int) {
	if retryMs <= 0 {
		retryMs = int(config.AtlasOutboundMinBackoff().Milliseconds())
	}
	c.outMu.Lock()
	c.outBackoffUntil = time.Now().Add(time.Duration(retryMs) * time.Millisecond)
	c.outMu.Unlock()
}

func buildPath(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	return base + "?" + v.Encode()
}

// GetSeries fetches /series with optional query params.
func (c *Client) GetSeries(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.Get(ctx, buildPath("/series", params))
}

// GetPlayers fetches /players with optional query params.
func (c *Client) GetPlayers(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.Get(ctx, buildPath("/players", params))
}

// GetTeams fetches /teams with optional query params.
func (c *Client) GetTeams(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.Get(ctx, buildPath("/teams", params))
}

// GetRosters fetches /rosters with optional query params.
func (c *Client) GetRosters(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.Get(ctx, buildPath("/rosters", params))
}

// getAllPages fetches all pages from a paginated endpoint and returns merged results.
// Follows Atlas pagination: take [0,50], skip [0,∞); stop when fewer than take
// items or empty array.
func (c *Client) getAllPages(ctx context.Context, path string, baseParams map[string]string) ([]byte, *RateLimit, error) {
	var all []json.RawMessage
	var lastRL *RateLimit
	for skip := 0; ; skip += config.PageSize() {
		params := make(map[string]string)
		for k, v := range baseParams {
			params[k] = v
		}
		params["skip"] = strconv.Itoa(skip)
		params["take"] = strconv.Itoa(config.PageSize())

		body, rl, err := c.Get(ctx, buildPath(path, params))
		if err != nil {
			return nil, rl, err
		}
		lastRL = rl

		var page []json.RawMessage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, rl, err
		}
		if config.Debug() {
			log.Printf("pagination: %s skip=%d -> %d items", path, skip, len(page))
		}
		all = append(all, page...)
		// Stop when fewer than take (includes empty array)
		if len(page) < config.PageSize() {
			break
		}
	}
	out, err := json.Marshal(all)
	if err != nil {
		return nil, lastRL, err
	}
	return out, lastRL, nil
}

// GetSeriesAll fetches all series matching params, paginating until complete.
func (c *Client) GetSeriesAll(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.getAllPages(ctx, "/series", params)
}

// GetRostersAll fetches all rosters matching params, paginating until complete.
func (c *Client) GetRostersAll(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.getAllPages(ctx, "/rosters", params)
}

// GetPlayersAll fetches all players matching params, paginating until complete.
func (c *Client) GetPlayersAll(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.getAllPages(ctx, "/players", params)
}

// GetTeamsAll fetches all teams matching params, paginating until complete.
func (c *Client) GetTeamsAll(ctx context.Context, params map[string]string) ([]byte, *RateLimit, error) {
	return c.getAllPages(ctx, "/teams", params)
}

// FilterIDIn formats filter=id<={ids} for the Atlas API.
// IDs are comma-separated in curly braces, e.g. filter=id<={1,2,3}.
func FilterIDIn(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	b := make([]byte, 0, 32)
	b = append(b, "id<={"...)
	for i, id := range ids {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendInt(b, int64(id), 10)
	}
	b = append(b, '}')
	return string(b)
}

func parseRateLimit(h http.Header) *RateLimit {
	rl := &RateLimit{}
	if v := h.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Limit = n
		}
	}
	if v := h.Get("X-RateLimit-Burst"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Burst = n
		}
	}
	if v := h.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Remaining = n
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.ResetMs = n
		}
	}
	return rl
}

func parseRetryAfter(s string) int {
	if s == "" {
		return 1000
	}
	ms, err := strconv.Atoi(s)
	if err != nil {
		return 1000
	}
	return ms
}
