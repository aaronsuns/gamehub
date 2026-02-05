.PHONY: run test integration-test loadtest build lint docker-build docker-run docker-test

# Server
run:
	go run ./cmd/server

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
