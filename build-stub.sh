#!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Set output directory
mkdir -p bin

echo "Starting stub build of CAZA OpenTelemetry AI Processor..."
echo "This build uses a stub WASM runtime (no wasmer-go dependency required)"

# Clean any old build artifacts
rm -f go.sum

# Validate Go version
if ! command -v go &> /dev/null; then
  echo "❌ Error: Go is not installed. Please install Go 1.23 or higher."
  exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c 3-)
if [ "$(echo "$GO_VERSION < 1.22" | bc -l 2>/dev/null)" == "1" ]; then
  echo "❌ Error: Go version $GO_VERSION detected, but 1.22 or higher is required."
  exit 1
fi

echo "✅ Go version $GO_VERSION verified."

# Download and update dependencies
echo "Downloading dependencies..."
go mod download || { echo "❌ Failed to download dependencies"; exit 1; }

echo "Updating go.mod and go.sum..."
go mod tidy || { echo "❌ Failed to tidy dependencies"; exit 1; }

# Explicitly get all the required dependencies except wasmer-go
echo "Explicitly downloading required packages..."
for pkg in \
  github.com/hashicorp/golang-lru/v2 \
  go.uber.org/zap \
  go.opentelemetry.io/collector/component \
  go.opentelemetry.io/collector/consumer \
  go.opentelemetry.io/collector/pdata \
  go.opentelemetry.io/collector/processor \
  go.opentelemetry.io/collector/processor/processorhelper \
  go.opentelemetry.io/collector/confmap \
  go.opentelemetry.io/collector/exporter \
  go.opentelemetry.io/collector/exporter/otlpexporter \
  go.opentelemetry.io/collector/otelcol \
  go.opentelemetry.io/collector/receiver \
  go.opentelemetry.io/collector/receiver/otlpreceiver
do
  echo "Downloading $pkg..."
  go get -v $pkg
done

# Final dependency check
echo "Final dependency check..."
go mod tidy || { echo "❌ Failed final dependency check"; exit 1; }

# Build the processor with stub implementation (no wasmer-go)
echo "Building processor with stub WASM runtime..."
CGO_ENABLED=0 go build -v -o bin/otel-ai-processor ./cmd/processor

# Check if build was successful
if [ $? -eq 0 ]; then
  echo ""
  echo "✅ Stub build completed successfully!"
  echo ""
  echo "The processor binary is located at: $(pwd)/bin/otel-ai-processor"
  echo ""
  echo "NOTE: This build includes a WASM runtime stub implementation which does not actually execute"
  echo "WASM models. Instead, it uses hardcoded default values for model responses."
  echo ""
  echo "To run the processor:"
  echo "  ./bin/otel-ai-processor --config=./config/config.yaml"
else
  echo ""
  echo "❌ Build failed."
  echo "Check the error messages above for more details."
fi