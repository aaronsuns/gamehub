.PHONY: run run-stress test integration-test loadtest stress stress-demo stress-demo-docker stop kill-8080 build lint docker-build docker-build-run docker-run docker-test test-endpoints

# ============================================================================
# CONFIGURATION - Adjust these values for demo/testing
# ============================================================================

# Server Configuration
PORT                    ?= 8080 # Port the server listens on (default: 8080)
GAMEHUB_DEBUG           ?=         # Enable debug logging (set to "1" to enable, shows pagination and other debug info)

# Rate Limiting Configuration
# Controls how many requests per IP address are allowed within a time window
GAMEHUB_INBOUND_RATE_LIMIT          ?= 120    # Max requests per IP per window (default: 120 requests/minute)
GAMEHUB_INBOUND_RATE_LIMIT_PER      ?= 1m     # Rate limit window duration (default: 1 minute)
GAMEHUB_INBOUND_RETRY_AFTER         ?= 60     # Retry-After header value (in seconds) sent with 429 responses (default: 60)
GAMEHUB_INBOUND_BUCKET_MAX_STALE    ?= 5m    # How long a rate limit bucket can be unused before eviction (default: 5 minutes)
GAMEHUB_INBOUND_BUCKET_EVICT_THRESHOLD ?= 100 # Threshold that triggers eviction cleanup (when total buckets > 100, removes stale buckets unused for GAMEHUB_INBOUND_BUCKET_MAX_STALE, default: 100)

# Caching Configuration
# Controls how long live context (teams/players in live series) is cached
GAMEHUB_LIVE_CACHE_TTL  ?= 10s     # Live context cache TTL (default: 10 seconds, reduces Atlas API calls)

# Atlas API Configuration
# Controls how the server interacts with the upstream Atlas API
GAMEHUB_PAGE_SIZE                    ?= 50    # Atlas API pagination page size (max: 50, default: 50)
GAMEHUB_ATLAS_CLIENT_TIMEOUT         ?= 30s   # HTTP client timeout for Atlas API requests (default: 30 seconds)
GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF   ?= 1s    # Minimum backoff time when Atlas returns 429 without Retry-After header (default: 1 second)

# Stress Test Configuration
# Controls the stress test tool behavior (used by 'make stress' command)
STRESS_URL         ?= http://localhost:8080    # Base URL to test (default: http://localhost:8080)
STRESS_PATH        ?= /players/live            # API endpoint path to stress test (default: /players/live)
STRESS_DURATION    ?= 5m                      # Duration to run continuously (e.g., 5m, 10m). Set to "0" to use request count mode instead
STRESS_N           ?= 0                       # Total requests to send (only used if STRESS_DURATION=0, default: 0 = use duration mode)
STRESS_CONCURRENCY ?= 4                       # Number of concurrent worker goroutines (default: 4 workers)
STRESS_DELAY       ?= 500ms                   # Delay between requests from each worker (default: 500ms = ~2 req/s per worker)
                                                # Rate calculation: 4 workers * 2 req/s = ~8 req/s = ~480 req/min (~4x the 120/min limit, hits limit after ~15s)
STRESS_TIMEOUT     ?= 30s                     # HTTP client timeout for stress test requests (default: 30 seconds)
STRESS_VERBOSE     ?=                         # Verbose output mode (set to "1" to see individual request details)
STRESS_PROGRESS    ?= 1                       # Show progress updates every 10 seconds during long runs (set to "0" to disable, default: 1)

# ============================================================================
# SERVER COMMANDS
# ============================================================================

# Run server with default configuration
run:
	GAMEHUB_INBOUND_RATE_LIMIT=$(GAMEHUB_INBOUND_RATE_LIMIT) \
	GAMEHUB_INBOUND_RATE_LIMIT_PER=$(GAMEHUB_INBOUND_RATE_LIMIT_PER) \
	GAMEHUB_LIVE_CACHE_TTL=$(GAMEHUB_LIVE_CACHE_TTL) \
	GAMEHUB_PAGE_SIZE=$(GAMEHUB_PAGE_SIZE) \
	GAMEHUB_ATLAS_CLIENT_TIMEOUT=$(GAMEHUB_ATLAS_CLIENT_TIMEOUT) \
	GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF=$(GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF) \
	GAMEHUB_DEBUG=$(GAMEHUB_DEBUG) \
	PORT=$(PORT) \
	go run ./cmd/server

# Run server with stress test configuration (smaller page size to trigger Atlas 429s)
run-stress:
	GAMEHUB_PAGE_SIZE=$(GAMEHUB_PAGE_SIZE) \
	GAMEHUB_INBOUND_RATE_LIMIT=$(GAMEHUB_INBOUND_RATE_LIMIT) \
	GAMEHUB_INBOUND_RATE_LIMIT_PER=$(GAMEHUB_INBOUND_RATE_LIMIT_PER) \
	go run ./cmd/server

# ============================================================================
# TESTING COMMANDS
# ============================================================================

# Unit tests
test:
	go test ./...

# Lint (runs in container, same as CI)
lint:
	docker run --rm -v "$$(pwd):/app" -w /app golangci/golangci-lint:latest golangci-lint run

# Integration test (requires ATLAS_API_KEY; skips if unset)
integration-test:
	go test ./internal/handlers/ -run TestIntegration -v

# Test all endpoints (server must be running)
test-endpoints:
	@./scripts/test-endpoints.sh

# ============================================================================
# STRESS TEST COMMANDS
# ============================================================================

# Simple load test (server must be running)
loadtest:
	go run ./cmd/loadtest -url $(STRESS_URL) -n $(STRESS_N) -delay $(STRESS_DELAY)

# Advanced stress test with concurrency (server must be running)
# Runs continuously for STRESS_DURATION (default: 5 minutes)
stress:
	STRESS_URL=$(STRESS_URL) \
	STRESS_PATH=$(STRESS_PATH) \
	STRESS_DURATION=$(STRESS_DURATION) \
	STRESS_N=$(STRESS_N) \
	STRESS_CONCURRENCY=$(STRESS_CONCURRENCY) \
	STRESS_DELAY=$(STRESS_DELAY) \
	STRESS_TIMEOUT=$(STRESS_TIMEOUT) \
	STRESS_VERBOSE=$(STRESS_VERBOSE) \
	STRESS_PROGRESS=$(STRESS_PROGRESS) \
	go run ./cmd/stresstest

# One-shot stress demo: starts server in background, runs stress test
stress-demo:
	@echo "Starting server..."
	@GAMEHUB_PAGE_SIZE=$(GAMEHUB_PAGE_SIZE) \
	GAMEHUB_INBOUND_RATE_LIMIT=$(GAMEHUB_INBOUND_RATE_LIMIT) \
	GAMEHUB_INBOUND_RATE_LIMIT_PER=$(GAMEHUB_INBOUND_RATE_LIMIT_PER) \
	go run ./cmd/server & SERVER_PID=$$!; \
	sleep 4; \
	echo ""; echo "  >>> Open http://localhost:$$(echo $(PORT) | tr -d '[:space:]')/monitor in your browser <<<"; echo ""; \
	sleep 2; \
	STRESS_URL=$(STRESS_URL) \
	STRESS_PATH=$(STRESS_PATH) \
	STRESS_DURATION=$(STRESS_DURATION) \
	STRESS_N=$(STRESS_N) \
	STRESS_CONCURRENCY=$(STRESS_CONCURRENCY) \
	STRESS_DELAY=$(STRESS_DELAY) \
	STRESS_PROGRESS=$(STRESS_PROGRESS) \
	go run ./cmd/stresstest; \
	echo ""; echo "Done. Server (PID $$SERVER_PID) still running. kill $$SERVER_PID to stop."

# Stress demo in Docker: server in container, stress test from host
stress-demo-docker: docker-build
	@if [ -z "$${ATLAS_API_KEY}" ]; then \
		echo "Error: ATLAS_API_KEY environment variable is not set"; \
		exit 1; \
	fi
	@docker stop gamehub-stress 2>/dev/null; docker rm gamehub-stress 2>/dev/null; true
	@echo "Starting container..."
	@PORT_VAL=$$(echo $(PORT) | tr -d '[:space:]'); \
	docker run -d \
		-e ATLAS_API_KEY="$${ATLAS_API_KEY}" \
		-e GAMEHUB_PAGE_SIZE=$(GAMEHUB_PAGE_SIZE) \
		-e GAMEHUB_INBOUND_RATE_LIMIT=$(GAMEHUB_INBOUND_RATE_LIMIT) \
		-e GAMEHUB_INBOUND_RATE_LIMIT_PER=$(GAMEHUB_INBOUND_RATE_LIMIT_PER) \
		-e GAMEHUB_LIVE_CACHE_TTL=$(GAMEHUB_LIVE_CACHE_TTL) \
		-p $$PORT_VAL:8080 \
		--name gamehub-stress \
		gamehub:latest
	@trap 'echo ""; echo "Stopping container..."; docker stop gamehub-stress 2>/dev/null; docker rm gamehub-stress 2>/dev/null; echo "Container stopped."; exit 130' INT TERM; \
	sleep 4; \
	echo ""; echo "  >>> Open http://localhost:$$(echo $(PORT) | tr -d '[:space:]')/monitor in your browser <<<"; echo ""; sleep 2; \
	STRESS_URL=$(STRESS_URL) \
	STRESS_PATH=$(STRESS_PATH) \
	STRESS_DURATION=$(STRESS_DURATION) \
	STRESS_N=$(STRESS_N) \
	STRESS_CONCURRENCY=$(STRESS_CONCURRENCY) \
	STRESS_DELAY=$(STRESS_DELAY) \
	STRESS_PROGRESS=$(STRESS_PROGRESS) \
	go run ./cmd/stresstest; \
	echo ""; echo "Stopping container..."; \
	docker stop gamehub-stress 2>/dev/null; docker rm gamehub-stress 2>/dev/null; \
	echo "Done. Container stopped."

# ============================================================================
# DOCKER COMMANDS
# ============================================================================

# Build Docker image
docker-build:
	docker build -t gamehub .

# Build and run Docker container
docker-build-run: docker-build docker-run

# Run Docker container (uses cached image, does not rebuild)
docker-run:
	@PORT_VAL=$$(echo $(PORT) | tr -d '[:space:]'); \
	docker run \
		-e ATLAS_API_KEY="$${ATLAS_API_KEY}" \
		-e GAMEHUB_PAGE_SIZE=$(GAMEHUB_PAGE_SIZE) \
		-e GAMEHUB_INBOUND_RATE_LIMIT=$(GAMEHUB_INBOUND_RATE_LIMIT) \
		-e GAMEHUB_INBOUND_RATE_LIMIT_PER=$(GAMEHUB_INBOUND_RATE_LIMIT_PER) \
		-e GAMEHUB_LIVE_CACHE_TTL=$(GAMEHUB_LIVE_CACHE_TTL) \
		-e GAMEHUB_ATLAS_CLIENT_TIMEOUT=$(GAMEHUB_ATLAS_CLIENT_TIMEOUT) \
		-e GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF=$(GAMEHUB_ATLAS_OUTBOUND_MIN_BACKOFF) \
		-p $$PORT_VAL:8080 \
		gamehub

# Docker test: build, run container on 8081, test, stop
docker-test: docker-build
	@echo "Starting container..."
	@docker run -d -e ATLAS_API_KEY="$${ATLAS_API_KEY}" -p 8081:8080 --name gamehub-test gamehub
	@sleep 2
	@GAMEHUB_DOCKERTEST_URL=http://localhost:8081 go run ./cmd/dockertest; EX=$$?; docker stop gamehub-test 2>/dev/null; docker rm gamehub-test 2>/dev/null; exit $$EX

# ============================================================================
# UTILITY COMMANDS
# ============================================================================

# Stop Docker stress/test containers
stop:
	@docker stop gamehub-stress gamehub-test 2>/dev/null; docker rm gamehub-stress gamehub-test 2>/dev/null; echo "Stopped."

# Free port 8080: stop containers and kill any process listening on 8080
kill-8080: stop
	@PIDS=$$(lsof -ti :$(PORT) 2>/dev/null); [ -n "$$PIDS" ] && kill $$PIDS && echo "Killed process on $(PORT)" || echo "Port $(PORT) is free"

# Build binary
build:
	go build -o bin/server ./cmd/server
