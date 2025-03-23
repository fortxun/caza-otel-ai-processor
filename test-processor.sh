#!/bin/bash

# Start the AI processor in the background
cd "$(dirname "$0")"

echo "Starting the AI processor with WASM support..."
./bin/otel-ai-processor-wasm --config=./config/config.yaml &
PROCESSOR_PID=$!

# Wait for processor to start
echo "Waiting for processor to start up..."
sleep 3

# Send test trace data
echo "Sending test trace data with otel-cli..."

# Test trace with error
otel-cli exec \
  --service "test-service" \
  --name "test-span-with-error" \
  --endpoint http://localhost:14318 \
  --attrs "operation.type=database,db.system=postgresql,error=true,message=Connection refused" \
  -- echo "Simulating operation with database error"

# Test trace with slow operation
otel-cli exec \
  --service "test-service" \
  --name "test-span-slow-operation" \
  --endpoint http://localhost:14318 \
  --attrs "operation.type=http,http.method=GET,duration.ms=750" \
  -- sleep 0.75 && echo "Simulating slow HTTP operation"

# Test normal operation trace
otel-cli exec \
  --service "test-service" \
  --name "test-span-normal" \
  --endpoint http://localhost:14318 \
  --attrs "operation.type=cache,cache.hit=true" \
  -- echo "Simulating normal cache operation"

# Wait a bit for processing to complete
echo "Waiting for processing to complete..."
sleep 3

# Kill the processor
echo "Stopping the AI processor..."
kill $PROCESSOR_PID

# Display log file with relevant WASM processing events
echo ""
echo "Processor log output - WASM model loading:"
echo "=========================================="
cat processor.log | grep -E "Loaded|model|runtime" | sort -u | head -10

echo ""
echo "Processor log output - Trace processing:"
echo "========================================"
cat processor.log | grep -E "test-span|trace|otel|process|WASM" | head -30

echo ""
echo "Processor log output - WASM function calls:"
echo "=========================================="
cat processor.log | grep -E "classify_error|sample_telemetry|extract_entities|AssemblyScript|invoke" | head -20