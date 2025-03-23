#!/bin/bash

# Script to run the WASM performance benchmarks
# This script:
# 1. Builds the WASM models
# 2. Builds the processor with fullwasm tag 
# 3. Runs the benchmarks comparing WASM vs Stub implementations
# 4. Generates a benchmark report

set -e

# Change to the project root directory
cd "$(dirname "$0")"

echo "📊 Starting WASM performance benchmark process..."

# Step 1: Build the WASM models if they don't exist
if [ ! -f "wasm-models/error-classifier/build/error-classifier.wasm" ] || \
   [ ! -f "wasm-models/importance-sampler/build/importance-sampler.wasm" ] || \
   [ ! -f "wasm-models/entity-extractor/build/entity-extractor.wasm" ]; then
  echo "📦 Building WASM models..."
  ./build-wasm.sh

  if [ $? -ne 0 ]; then
    echo "❌ WASM model build failed. Aborting."
    exit 1
  fi
fi

# Step 2: Build the processor with fullwasm support
echo "🔨 Building processor with WASM support..."
./build-full-wasm.sh

if [ $? -ne 0 ]; then
  echo "❌ Processor build failed. Aborting."
  exit 1
fi

# Step 3: Run the benchmarks
echo "📊 Running WASM performance benchmarks..."

# Create a results directory
mkdir -p benchmark-results

# Get the current date and time for the filename
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_FILE="benchmark-results/wasm_benchmark_${TIMESTAMP}.txt"
FORMATTED_RESULTS_FILE="benchmark-results/wasm_benchmark_report_${TIMESTAMP}.md"

# Set the FULLWASM_TEST environment variable
export FULLWASM_TEST=1

# Run the benchmarks and save the results
echo "Running benchmarks (this may take several minutes)..."
cd pkg/processor/tests
go test -tags=fullwasm -run=^$ -bench=BenchmarkWasm -benchmem -count=3 | tee ../../../${RESULTS_FILE}

# Exit if tests failed
if [ $? -ne 0 ]; then
  echo ""
  echo "❌ Benchmark tests failed."
  exit 1
fi

cd ../../..

# Step 4: Format the results into a more readable report
echo "Generating benchmark report..."

cat > ${FORMATTED_RESULTS_FILE} << EOF
# WASM vs Stub Performance Benchmark Report

Date: $(date +"%Y-%m-%d %H:%M:%S")

This report compares the performance of the WASM implementation vs the stub implementation
of the OpenTelemetry AI Processor for various operations.

## Summary

The benchmark tests compare:
- Error classification performance (WASM vs Stub)
- Telemetry sampling performance (WASM vs Stub)
- Entity extraction performance (WASM vs Stub)
- Full processing pipeline performance (WASM vs Stub)

## Raw Benchmark Results

\`\`\`
$(cat ${RESULTS_FILE})
\`\`\`

## Analysis

### Error Classification
$(grep -A 5 "BenchmarkWasmVsStub_ErrorClassification" ${RESULTS_FILE} | tail -n 2 | awk '{print "- " $1 ": " $3 " ops/sec, " $4 " ns/op, " $5 " B/op, " $6 " allocs/op"}')

### Telemetry Sampling
$(grep -A 5 "BenchmarkWasmVsStub_SampleTelemetry" ${RESULTS_FILE} | tail -n 2 | awk '{print "- " $1 ": " $3 " ops/sec, " $4 " ns/op, " $5 " B/op, " $6 " allocs/op"}')

### Entity Extraction
$(grep -A 5 "BenchmarkWasmVsStub_EntityExtraction" ${RESULTS_FILE} | tail -n 2 | awk '{print "- " $1 ": " $3 " ops/sec, " $4 " ns/op, " $5 " B/op, " $6 " allocs/op"}')

### Full Pipeline
$(grep -A 5 "BenchmarkWasmVsStub_FullPipeline" ${RESULTS_FILE} | tail -n 2 | awk '{print "- " $1 ": " $3 " ops/sec, " $4 " ns/op, " $5 " B/op, " $6 " allocs/op"}')

## Conclusion

This benchmark provides a comparison of resource usage between the WASM and stub implementations.
The WASM implementation is generally expected to be slightly slower but provides the flexibility
of runtime-loaded AI models without recompilation.

For production use, consider the performance tradeoffs against the benefits of dynamic model loading.
EOF

echo ""
echo "✅ Benchmark tests completed successfully!"
echo "📈 Benchmark report saved to: ${FORMATTED_RESULTS_FILE}"
echo ""
echo "Next steps:"
echo "1. Review the benchmark report to understand performance differences"
echo "2. Run the full WASM integration tests with: ./test-fullwasm-integration.sh"
echo "3. Consider production deployment configurations based on performance requirements"