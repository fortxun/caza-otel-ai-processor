#!/bin/bash

# Script to run the comprehensive WASM integration tests
# This script:
# 1. Builds the WASM models
# 2. Builds the processor with fullwasm tag
# 3. Runs the integration tests with FULLWASM_TEST=1

set -e

# Change to the project root directory
cd "$(dirname "$0")"

echo "🧪 Starting comprehensive WASM integration testing process..."

# Step 1: Build the WASM models
echo "📦 Building WASM models..."
./build-wasm.sh

if [ $? -ne 0 ]; then
  echo "❌ WASM model build failed. Aborting."
  exit 1
fi

# Step 2: Build the processor with fullwasm support
echo "🔨 Building processor with WASM support..."
./build-full-wasm.sh

if [ $? -ne 0 ]; then
  echo "❌ Processor build failed. Aborting."
  exit 1
fi

# Step 3: Run the integration tests
echo "🧪 Running WASM integration tests..."
cd pkg/processor/tests

# Set the FULLWASM_TEST environment variable to enable the WASM tests
export FULLWASM_TEST=1

# Run the test with verbose output
go test -v -tags=fullwasm -run TestWasmIntegration

# Check the test result
if [ $? -eq 0 ]; then
  echo ""
  echo "✅ WASM integration tests completed successfully!"
else
  echo ""
  echo "❌ WASM integration tests failed."
  exit 1
fi

echo ""
echo "📊 All tests completed. The OpenTelemetry AI Processor with WASM integration is working correctly."
echo "🔍 Next steps: Consider running performance benchmarks or deploying to a test environment."