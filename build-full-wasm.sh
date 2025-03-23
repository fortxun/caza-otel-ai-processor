#\!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Set output directory
mkdir -p bin

echo "🧠 Building CAZA OpenTelemetry AI Processor with full WASM support..."

# First, build the WASM models
./build-wasm.sh

if [ $? -ne 0 ]; then
  echo "❌ WASM model build failed. Aborting."
  exit 1
fi

# Validate Go version
if \! command -v go &> /dev/null; then
  echo "❌ Error: Go is not installed. Please install Go 1.23 or higher."
  exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c 3-)
if [ "$(echo "$GO_VERSION < 1.23" | bc -l)" -eq 1 ]; then
  echo "❌ Error: Go version $GO_VERSION detected, but 1.23 or higher is required."
  exit 1
fi

echo "✅ Go version $GO_VERSION verified."

# Download and update dependencies
echo "Downloading dependencies..."
go mod download || { echo "❌ Failed to download dependencies"; exit 1; }

echo "Updating go.mod and go.sum..."
go mod tidy || { echo "❌ Failed to tidy dependencies"; exit 1; }

# Explicitly get wasmer-go dependency
echo "Downloading wasmer-go dependency..."
go get -v github.com/wasmerio/wasmer-go/wasmer

# Build the processor with wasmer-go support (which requires CGO)
# IMPORTANT: Use the fullwasm build tag 
echo "Building processor with WASM support (fullwasm tag)..."
CGO_ENABLED=1 go build -v -tags=fullwasm -o bin/otel-ai-processor-wasm ./cmd/processor

# Check if build was successful
if [ $? -eq 0 ]; then
  echo ""
  echo "✅ Build completed successfully\!"
  echo ""
  echo "The processor binary with WASM support is located at: $(pwd)/bin/otel-ai-processor-wasm"
  echo ""
  echo "To run the processor:"
  echo "  ./bin/otel-ai-processor-wasm --config=./config/config.yaml"
else
  echo ""
  echo "❌ Build failed."
  echo "Check the error messages above for more details."
fi
