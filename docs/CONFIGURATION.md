# Configuration Guide

All configuration is centralized in the `Makefile` for easy adjustment during demos.

## Quick Configuration

Edit the top of `Makefile` to adjust any setting:

```makefile
# Rate Limiting Configuration
GAMEHUB_INBOUND_RATE_LIMIT          ?= 60    # Requests per IP per window
GAMEHUB_INBOUND_RATE_LIMIT_PER      ?= 1m    # Rate limit window

# Stress Test Configuration
STRESS_N           ?= 100      # Total requests
STRESS_CONCURRENCY ?= 10       # Concurrent workers
STRESS_DELAY       ?= 50ms     # Delay between requests
STRESS_PATH        ?= /players/live  # Endpoint to test
```

## Configuration Categories

### Server Configuration
- `PORT` - Server port (default: 8080)
- `GAMEHUB_DEBUG` - Enable debug logging (set to `1` to enable)

### Rate Limiting
- `GAMEHUB_INBOUND_RATE_LIMIT` - Max requests per IP per window (default: 60)
- `GAMEHUB_INBOUND_RATE_LIMIT_PER` - Rate limit window (default: 1m)
- `GAMEHUB_INBOUND_RETRY_AFTER` - Retry-After header value on 429 (default: 60)
- `GAMEHUB_INBOUND_BUCKET_MAX_STALE` - Eviction threshold for stale buckets (default: 5m)
- `GAMEHUB_INBOUND_BUCKET_EVICT_THRESHOLD` - Bucket count threshold for eviction (default: 100)

### Caching
- `GAMEHUB_LIVE_CACHE_TTL` - Live context cache TTL (default: 10s)

### Atlas API
- `GAMEHUB_PAGE_SIZE` - Atlas pagination page size (default: 50)
- `GAMEHUB_ATLAS_CLIENT_TIMEOUT` - HTTP client timeout (default: 30s)
- `GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF` - Min backoff on 429 (default: 1s)

### Stress Test
- `STRESS_URL` - Base URL (default: http://localhost:8080)
- `STRESS_PATH` - API path to test (default: /players/live)
- `STRESS_DURATION` - Duration to run continuously (default: 5m). Set to `0` to use request count mode
- `STRESS_N` - Total requests (default: 0, only used if STRESS_DURATION=0)
- `STRESS_CONCURRENCY` - Concurrent workers (default: 10)
- `STRESS_DELAY` - Delay between requests (default: 50ms)
- `STRESS_TIMEOUT` - HTTP client timeout (default: 30s)
- `STRESS_VERBOSE` - Verbose output (set to `1` to enable)
- `STRESS_PROGRESS` - Show progress updates every 10s (default: 1)

## Usage Examples

### Override Configuration at Runtime

```bash
# Run with custom rate limit
make run GAMEHUB_INBOUND_RATE_LIMIT=30

# Stress test with more requests
make stress STRESS_N=200 STRESS_CONCURRENCY=20

# Test different endpoint
make stress STRESS_PATH=/series/live

# Run stress demo with custom config
make stress-demo STRESS_N=150 GAMEHUB_INBOUND_RATE_LIMIT=50
```

### Common Demo Scenarios

**Scenario 1: Quick Rate Limit Demo**
```bash
# Low rate limit for quick demo
make run GAMEHUB_INBOUND_RATE_LIMIT=10
# In another terminal
make stress STRESS_N=20 STRESS_DELAY=100ms
```

**Scenario 2: High Concurrency Test**
```bash
make run
make stress STRESS_DURATION=2m STRESS_CONCURRENCY=50 STRESS_DELAY=10ms
```

**Scenario 2b: Long Duration Test (several minutes)**
```bash
make run
make stress STRESS_DURATION=10m STRESS_CONCURRENCY=10 STRESS_DELAY=100ms
```

**Scenario 3: Test Different Endpoints**
```bash
make run
make stress STRESS_PATH=/series/live STRESS_N=50
make stress STRESS_PATH=/teams/live STRESS_N=50
make stress STRESS_PATH=/players/live STRESS_N=50
```

**Scenario 4: Trigger Atlas Rate Limits**
```bash
# Small page size triggers more Atlas API calls
make run-stress GAMEHUB_PAGE_SIZE=5
make stress STRESS_PATH=/players/live STRESS_N=100
```

## All-in-One Commands

The Makefile includes convenient commands that use all configurations:

- `make run` - Run server with all configs
- `make stress` - Run stress test (server must be running)
- `make stress-demo` - Start server + run stress test
- `make stress-demo-docker` - Docker version of stress demo
- `make docker-run` - Run in Docker with all configs

All commands automatically use the configuration values from the top of the Makefile.
