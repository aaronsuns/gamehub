# GameHub — Atlas API Wrapper

HTTP server in Go that wraps the Atlas esports data API, exposing live series, players, and teams.

## Endpoints

- `GET /series/live` — Live/ongoing series
- `GET /players/live` — Players in live series
- `GET /teams/live` — Teams in live series

## Project Layout

- `cmd/server` — main HTTP server
- `cmd/loadtest` — load test tool to exercise inbound rate limiting
- `cmd/dockertest` — Docker test client (used by `make docker-test`)
- `internal/atlas` — Atlas API client with pagination
- `internal/handlers` — HTTP handlers
- `internal/live` — live context derivation and caching
- `internal/middleware` — inbound rate limiting
- `internal/config` — constants (page size, rate limits, cache TTL)

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

## Docker

Ensure Docker is running, then:

```bash
make docker-build
make docker-run
```

Or run manually:

```bash
docker build -t gamehub .
docker run -e ATLAS_API_KEY="$ATLAS_API_KEY" -p 8080:8080 gamehub
```

The server listens on port 8080 inside the container.

**Docker test** — build, run container, hit all 3 endpoints with formatted output, then stop:

```bash
make docker-test
```

## Testing

```bash
make test          # Unit tests
make lint          # Lint (runs in Docker, same as CI)
make loadtest      # Load test (server must be running; start with 'make run' in another terminal)
# Expect: first ~60 requests return 200, rest return 429
```

CI runs on push/PR to `main`: lint and unit tests in a container.

### Integration tests (real API)

```bash
make integration-test
```

Runs against the real Atlas API, verifies HTTP 200 and valid JSON, writes responses to a local output folder. Skipped if `ATLAS_API_KEY` is unset.

Set `GAMEHUB_DEBUG=1` to enable debug output (e.g. pagination requests).
