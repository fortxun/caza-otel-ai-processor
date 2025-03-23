.PHONY: build test clean docker run

# Build settings
BINARY_NAME=otel-ai-processor
DOCKER_IMAGE=caza/otel-ai-processor

# Go settings
GO=go
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GO_BUILD_FLAGS=-v

# Directories
CMD_DIR=./cmd/processor
BUILD_DIR=./build
MODELS_DIR=./models

# Default target
all: build

# Build the binary
build:
	$(GO) build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-cover:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run unit tests
test-unit:
	$(GO) test -v ./... -run "^Test" -short

# Run integration tests
test-integration:
	$(GO) test -v ./... -run "^TestProcessor"
	
# Run benchmarks
test-bench:
	$(GO) test ./... -bench=. -benchmem

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	$(GO) clean

# Build Docker image
docker:
	docker build -t $(DOCKER_IMAGE):latest .

# Run locally
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Run with Docker
docker-run:
	docker run -p 4317:4317 -p 4318:4318 -p 13133:13133 \
		-v $(PWD)/config:/config \
		-v $(PWD)/models:/models \
		$(DOCKER_IMAGE):latest --config=/config/config.yaml

# Download dependencies
deps:
	$(GO) mod download

# Tidy dependencies
tidy:
	$(GO) mod tidy

# Lint code
lint:
	golangci-lint run

# Generate placeholder WASM models for testing
placeholders:
	@echo "Creating placeholder WASM models (Not implemented yet)"
	@echo "For actual development, real WASM models should be created"

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker       - Build Docker image"
	@echo "  run          - Run locally"
	@echo "  docker-run   - Run with Docker"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy dependencies"
	@echo "  lint         - Lint code"
	@echo "  placeholders - Generate placeholder WASM models for testing"
	@echo "  help         - Show this help"