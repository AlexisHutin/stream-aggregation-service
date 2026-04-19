SHELL := /bin/bash
ENV_FILE := .env
TEST_ENV_FILE := .env.test
SMOCKER_PORT := 8083
SMOCKER_API_PORT := 8082
VENOM := $(shell command -v venom 2>/dev/null || echo $(HOME)/.local/bin/venom)

suite = **/*.venom.yml
service_endpoint = http://localhost:8080
mock_server = http://localhost:8083

.PHONY: run
# Run the service in development mode with auto-reload via reflex.
run:
	@if [ -f $(ENV_FILE) ]; then \
		set -a; source $(ENV_FILE); set +a; \
	fi; \
	reflex -c reflex.conf

.PHONY: build
# Build the binary in ./build.
build:
	go build -o ./build/stream-aggregation-service main.go

.PHONY: lint
# Run static analysis with golangci-lint.
lint:
	golangci-lint run --config .golangci.yml

.PHONY: test
# Run all unit tests and generate a coverage profile.
test:
	go test -v ./... -coverprofile=./build/coverage.txt

.PHONY: integration-dependencies
# Prepare integration dependencies: Venom binary and smocker container.
integration-dependencies:	
	@if [ ! -x "$(VENOM)" ]; then \
		echo "Installing venom..."; \
		mkdir -p "$(dir $(VENOM))"; \
		curl https://github.com/ovh/venom/releases/download/v1.2.0/venom.linux-amd64 -L -o "$(VENOM)"; \
		chmod +x "$(VENOM)"; \
		"$(VENOM)" -h >/dev/null; \
	else \
		echo "venom already installed at $(VENOM)"; \
	fi
	@if docker ps -a --format '{{.Names}}' | grep -Fxq smocker; then \
		echo "Removing existing smocker container..."; \
		docker rm -f smocker >/dev/null; \
	fi
	docker run -d \
		--restart=always \
		-p $(SMOCKER_API_PORT):8080 \
		-p $(SMOCKER_PORT):8081 \
		--name smocker \
		ghcr.io/smocker-dev/smocker

.PHONY: integration
# Execute integration test suites with Venom.
integration:
	NO_GELF=true "$(VENOM)" run './tests/venom/$(suite)' \
		-v \
		--format=xml \
		--output-dir=./build \
		--var service_endpoint=$(service_endpoint) \
		--var mock_server=$(mock_server)

#== DOCKER ==#
.PHONY: run-docker
# Run the service container locally.
run-docker:
	docker run --env-file $(ENV_FILE) -v $(PWD)/config.json:/config.json:ro -p 8080:8080 stream-aggregation-service:latest

.PHONY: build-docker
# Build the Docker image.
build-docker:
	docker build -t stream-aggregation-service:latest .

#== UTILITY ==#
.PHONY: coverage
# Generate an HTML coverage report from build artifacts.
coverage:
	go tool cover -html=./build/coverage.out -o ./build/coverage.html

.PHONY: clean
# Remove integration container and clean build artifacts.
clean:
	@echo "Removing smocker container..."; \
	docker rm -f smocker >/dev/null; 
	@echo "Cleaning build directory..."; \
	rm -rf ./build/*

.PHONY: ci
# Run local CI flow: build, prepare integration deps, run service, run integration tests.
ci: build integration-dependencies
	@set -euo pipefail; \
	if [ -f $(TEST_ENV_FILE) ]; then \
		set -a; source $(TEST_ENV_FILE); set +a; \
	fi; \
	./build/stream-aggregation-service & \
	SERVICE_PID=$$!; \
	trap 'kill $$SERVICE_PID >/dev/null 2>&1 || true' EXIT; \
	echo "Started stream-aggregation-service (pid=$$SERVICE_PID)"; \
	$(MAKE) integration