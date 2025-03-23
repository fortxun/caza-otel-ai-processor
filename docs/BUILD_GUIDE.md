# Building the CAZA OpenTelemetry AI Processor

This document provides comprehensive instructions for building the CAZA OpenTelemetry AI Processor.

## Prerequisites

- Go 1.23 or higher (required for OpenTelemetry v0.95.0+)
- Node.js and npm (for building WASM models)
- Docker (optional, for container builds)

## Build Options

We provide several build methods depending on your environment:

### Option 1: Stub Build (Recommended for most users)

The most reliable approach using a WASM stub implementation that doesn't require wasmer-go:

```bash
# Build with WASM stub implementation (no wasmer-go dependency)
./build-stub.sh
```

This script will:
1. Check for Go 1.23+ installation
2. Build the processor with a stub WASM runtime implementation
3. The stub implementation mimics the AI models with hardcoded responses

**Note:** The stub version doesn't actually execute WASM models, but provides realistic mock responses for testing and development.

### Option 2: Using Docker with Stub Implementation

If you prefer using Docker with the stub implementation:

```bash
# Build using Docker with stub implementation
docker build -t fortxun/caza-otel-ai-processor:latest -f Dockerfile.stub .
```

### Option 3: Full Build with WASM Support

For a full build with actual WASM model execution (requires wasmer-go):

```bash
# First build the WASM models
./build-wasm.sh

# Then build the full processor with WASM support
CGO_ENABLED=0 go build -tags fullwasm -o ./bin/otel-ai-processor ./cmd/processor
```

### Option 4: Manual Build

For more control over the build process:

```bash
# 1. Clean any old build artifacts
rm -f go.sum

# 2. Update dependencies
go mod download
go mod tidy

# 3. Build the processor (stub implementation)
CGO_ENABLED=0 go build -o ./bin/otel-ai-processor ./cmd/processor

# OR for full WASM support:
CGO_ENABLED=0 go build -tags fullwasm -o ./bin/otel-ai-processor ./cmd/processor
```

## Key Dependencies

The processor uses these major dependencies:

- Go OpenTelemetry SDK v0.95.0 or higher
- wasmer-go for WASM model execution
- golang-lru for caching

## Dependency Notes

The OpenTelemetry packages have undergone significant changes:
- `go.opentelemetry.io/collector/model` is deprecated, use `pdata` instead
- The service package was replaced by the `otelcol` package in newer versions
- Component factories have changed their signatures
- Newer versions require Go 1.23 or higher

## Building WASM Models

If you need to rebuild the WASM models:

```bash
# Build all WASM models (requires Node.js and npm)
./build-wasm.sh
```

This will build the models and place them in the `./models` directory.

## Common Issues and Troubleshooting

1. **Missing Dependencies / go.sum Issues**:
   ```
   missing go.sum entry for module providing package...
   ```
   
   **Solution**: Use the direct-build.sh script which will explicitly download all dependencies:
   ```bash
   ./direct-build.sh
   ```

2. **Incompatible Go Version**: Ensure you're using Go 1.23+ as required by OpenTelemetry dependencies.
   
   **Solution**: Update your Go installation:
   ```bash
   # Check your current Go version
   go version
   
   # Install newer Go version if needed
   # (Instructions vary by OS - see golang.org/doc/install)
   ```

3. **Wasmer-Go Compatibility Issues**:
   ```
   undefined: wasmer.Instance
   field and method with the same name
   ```
   
   **Solution**: We've fixed these by renaming fields and adding a replace directive for wasmer-go in the go.mod file.

4. **Docker Build Issues**:
   If the Docker build fails, try the simplified Dockerfile:
   ```bash
   docker build -t fortxun/caza-otel-ai-processor:latest -f Dockerfile.simple .
   ```

5. **Import Path Changes**: OpenTelemetry packages have reorganized their structure:
   - Replace `go.opentelemetry.io/collector/model` with `go.opentelemetry.io/collector/pdata`  
   - Update service-related imports to use `otelcol` package

6. **CGO Required Errors**: If you see errors related to CGO:
   
   **Solution**: Explicitly disable CGO:
   ```bash
   CGO_ENABLED=0 go build -o ./bin/otel-ai-processor ./cmd/processor
   ```

## Running the Processor

After building:

```bash
# Run the binary directly
./bin/otel-ai-processor --config=./config/config.yaml

# Or with Docker
docker run -p 4317:4317 -p 4318:4318 \
  -v $(pwd)/config:/config \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

## Testing the Processor

Use the included tests:

```bash
# Run all tests
go test -v ./...

# Run benchmarks
go test -bench=. ./pkg/processor/tests
```