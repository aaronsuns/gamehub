# GameHub — Atlas API Wrapper

Go HTTP server (Gin) that wraps the Abios Atlas esports data API. Exposes live series, players, and teams with inbound rate limiting, a full live snapshot cache (series/players/teams), and observability (metrics dashboard + JSON stats).

## Architecture

- **Stack:** Gin router, per-IP fixed-window rate limit, TTL cache for a single live snapshot (all three `/live` endpoints served from cache when valid).
- **Docs:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — request flow, live snapshot, rate limit, outbound backoff.
- **Diagram:** [Figma — GameHub architecture board](https://www.figma.com/board/qyMmw0lEvQEb7FXbasHSou/gamehub?node-id=0-1&p=f&t=YfOwNnrzZFpcZpaN-0)

## Endpoints

- `GET /health` — Health check for liveness/readiness probes (no rate limit)
- `GET /monitor` — HTML dashboard with metrics and config (no rate limit)
- `GET /stats` — JSON metrics for polling (no rate limit)
- `GET /series/live` — Live/ongoing series (from cache when valid)
- `GET /players/live` — Players in live series (from cache when valid)
- `GET /teams/live` — Teams in live series (from cache when valid)

## Project Layout

- `cmd/server` — main HTTP server (Gin)
- `cmd/stresstest` — configurable stress test (duration or count, concurrency, delay)
- `cmd/dockertest` — Docker test client (used by `make docker-test`)
- `internal/atlas` — Atlas API client with pagination and outbound backoff
- `internal/handlers` — HTTP handlers
- `internal/live` — live snapshot loading and TTL cache (series + players + teams)
- `internal/metrics` — counters, history, monitor HTML, `/stats`
- `internal/middleware` — inbound fixed-window rate limiter
- `internal/config` — env-based config (rate limits, cache TTL, etc.)

## API Key Setup

### 1. Store the API key in Keychain

```bash
# Add the key to your login keychain (you'll be prompted for your Mac password)
security add-generic-password -a "$USER" -s "gamehub-api-key" -w "YOUR_API_KEY_HERE" -T ""
```

### 2. Load the key as an environment variable

```bash
# One-time export for current session
export ATLAS_API_KEY=$(security find-generic-password -a "$USER" -s "gamehub-api-key" -w)
```

## Running the Server

```bash
make run
```

Or build and run the binary:

```bash
make build
./bin/server
```

Server prints the monitor URL on startup (e.g. http://localhost:8080/monitor).

## Docker

Ensure Docker is running and `ATLAS_API_KEY` is set. All config (rate limit, cache TTL, etc.) is passed via env; see Makefile for variables.

```bash
make docker-build
make docker-run
```

Or build and run in one step:

```bash
make docker-build-run
```

**Note:** `make docker-run` uses the existing image; run `make docker-build` first after code changes.

**Docker test** — build, run container, hit endpoints, then stop:

```bash
make docker-test
```

## Testing

```bash
make test          # Unit tests
make lint          # Lint (Docker, same as CI)
make quality       # lint + test
make test-endpoints   # curl /health, /series/live, /players/live, /teams/live (server must be running)
```

CI runs on push/PR to `main`: lint and unit tests.

### Monitor and stress test

Free port 8080 if needed, then run server + stress test in Docker:

```bash
make stress-demo-docker
```

Uses the cached Docker image (run `make docker-build` first if you changed code). Opens the container, prints the monitor URL, runs the stress test. Open http://localhost:8080/monitor to see metrics, config, and the “live snapshot loads” counter. Use `make stop` to stop the container, `make kill-8080` to free the port.

See [docs/STRESS_TEST_RESULT.md](docs/STRESS_TEST_RESULT.md) for test configuration and how to interpret the graphs.

### Integration tests (real API)

```bash
make integration-test
```

Runs against the real Atlas API; skipped if `ATLAS_API_KEY` is unset. Set `GAMEHUB_DEBUG=1` for debug output.

### Configuration (environment variables)

Key variables (defaults in code; override via env or Makefile):

| Variable | Default | Description |
|----------|---------|-------------|
| `ATLAS_API_KEY` | (required) | Atlas API secret key |
| `PORT` | 8080 | Server port |
| `GAMEHUB_INBOUND_RATE_LIMIT` | 120 | Max requests per IP per window (fixed window) |
| `GAMEHUB_INBOUND_RATE_LIMIT_PER` | 1m | Rate limit window |
| `GAMEHUB_LIVE_CACHE_TTL` | 10s | Live snapshot cache TTL (series + players + teams) |
| `GAMEHUB_PAGE_SIZE` | 50 | Atlas pagination page size |
| `GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF` | 1s | Min backoff on Atlas 429 when Retry-After is missing |
| `GAMEHUB_DEBUG` | (unset) | Enable debug logging |

Full list and stress-test options are in the **Makefile** (top section). Use `make run` or `make stress-demo-docker` with env vars set there for demos.

## Quick Reference

**Start server:**
```bash
export ATLAS_API_KEY="your-key"
make run
```

**Test endpoints:**
```bash
curl http://localhost:8080/health
curl http://localhost:8080/series/live
curl http://localhost:8080/players/live
curl http://localhost:8080/teams/live
```

**View metrics:**
- Dashboard: http://localhost:8080/monitor
- JSON API: http://localhost:8080/stats

**Stress test (server must be running):**
```bash
make stress
# Or with custom duration: STRESS_DURATION=1m make stress
```
