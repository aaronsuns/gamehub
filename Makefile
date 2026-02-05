.PHONY: run run-stress test integration-test loadtest loadtest-stress stress-demo stress-demo-docker stop kill-8080 build lint docker-build docker-run docker-test

# Stress test config (override: make loadtest-stress STRESS_N=300 STRESS_DELAY=500ms STRESS_PAGE_SIZE=25)
STRESS_N          ?= 360
STRESS_DELAY      ?= 250ms
STRESS_PAGE_SIZE  ?= 10

# Server
run:
	go run ./cmd/server

# Server for stress test. Use with make loadtest-stress.
run-stress:
	GAMEHUB_PAGE_SIZE=$(STRESS_PAGE_SIZE) go run ./cmd/server

# Tests
test:
	go test ./...

# Lint (runs in container, same as CI)
lint:
	docker run --rm -v "$$(pwd):/app" -w /app golangci/golangci-lint:latest golangci-lint run

# Integration test (requires ATLAS_API_KEY; skips if unset)
integration-test:
	go test ./internal/handlers/ -run TestIntegration -v

# Load test (server must be running; start with 'make run' in another terminal)
loadtest:
	go run ./cmd/loadtest -url http://localhost:8080 -n 80

# Stress test. Run server with 'make run-stress' first, open /monitor, then run this.
loadtest-stress:
	go run ./cmd/loadtest -url http://localhost:8080 -path /players/live -n $(STRESS_N) -delay $(STRESS_DELAY)

# One-shot demo: starts server in background, runs loadtest. Open http://localhost:8080/monitor first.
stress-demo:
	@echo "Starting server (GAMEHUB_PAGE_SIZE=$(STRESS_PAGE_SIZE))..."; \
	GAMEHUB_PAGE_SIZE=$(STRESS_PAGE_SIZE) go run ./cmd/server & SERVER_PID=$$!; \
	sleep 4; \
	echo ""; echo "  >>> Open http://localhost:8080/monitor in your browser <<<"; echo ""; \
	sleep 2; \
	go run ./cmd/loadtest -url http://localhost:8080 -path /players/live -n $(STRESS_N) -delay $(STRESS_DELAY); \
	echo ""; echo "Done. Server (PID $$SERVER_PID) still running. kill $$SERVER_PID to stop."

# Stress demo in Docker: server in container, loadtest from host. Run 'make stop' to stop.
stress-demo-docker: docker-build
	@docker stop gamehub-stress 2>/dev/null; docker rm gamehub-stress 2>/dev/null; true
	@echo "Starting container (GAMEHUB_PAGE_SIZE=$(STRESS_PAGE_SIZE))..."
	@docker run -d -e ATLAS_API_KEY="$${ATLAS_API_KEY}" -e GAMEHUB_PAGE_SIZE=$(STRESS_PAGE_SIZE) -p 8080:8080 --name gamehub-stress gamehub
	@sleep 4
	@echo ""; echo "  >>> Open http://localhost:8080/monitor in your browser <<<"; echo ""; sleep 2
	@go run ./cmd/loadtest -url http://localhost:8080 -path /players/live -n $(STRESS_N) -delay $(STRESS_DELAY)
	@echo ""; echo "Done. Container gamehub-stress still running. make stop to stop."

# Stop Docker stress/test containers
stop:
	@docker stop gamehub-stress gamehub-test 2>/dev/null; docker rm gamehub-stress gamehub-test 2>/dev/null; echo "Stopped."

# Free port 8080: stop containers and kill any process listening on 8080
kill-8080: stop
	@PIDS=$$(lsof -ti :8080 2>/dev/null); [ -n "$$PIDS" ] && kill $$PIDS && echo "Killed process on 8080" || echo "Port 8080 is free"

# Build
build:
	go build -o bin/server ./cmd/server

# Docker
docker-build:
	docker build -t gamehub .

docker-run:
	docker run -e ATLAS_API_KEY="$${ATLAS_API_KEY}" -p 8080:8080 gamehub

# Docker test: build, run container on 8081 (avoids conflict with make run on 8080), test, stop
docker-test: docker-build
	@echo "Starting container..."
	@docker run -d -e ATLAS_API_KEY="$${ATLAS_API_KEY}" -p 8081:8080 --name gamehub-test gamehub
	@sleep 2
	@GAMEHUB_DOCKERTEST_URL=http://localhost:8081 go run ./cmd/dockertest; EX=$$?; docker stop gamehub-test 2>/dev/null; docker rm gamehub-test 2>/dev/null; exit $$EX
