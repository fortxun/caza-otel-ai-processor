#!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Check Go version
GO_VERSION=$(go version | grep -oP 'go\d+\.\d+' | grep -oP '\d+\.\d+')
REQUIRED_VERSION="1.23"

if (( $(echo "$GO_VERSION < $REQUIRED_VERSION" | bc -l) )); then
  echo "Error: Go version $GO_VERSION detected, but version $REQUIRED_VERSION or higher is required"
  echo "Please upgrade Go before continuing: https://golang.org/doc/install"
  exit 1
fi

# Create bin directory if it doesn't exist
mkdir -p ./bin

echo "Building CAZA OpenTelemetry AI Processor locally"
echo "This may take a few minutes..."

# Update dependencies
echo "Updating dependencies..."
go mod download && go mod tidy

# Build the processor
echo "Building processor..."
go build -o ./bin/otel-ai-processor ./cmd/processor

# Check if the build was successful
if [ $? -eq 0 ]; then
  echo "✅ Build completed successfully!"
  echo ""
  echo "To run the processor, use:"
  echo "./bin/otel-ai-processor --config=./config/config.yaml"
else
  echo "❌ Build failed."
  echo "Check the build logs above for errors."
  echo "For more detailed troubleshooting, see the BUILD_GUIDE.md file."
fi