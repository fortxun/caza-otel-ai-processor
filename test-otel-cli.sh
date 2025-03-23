#!/bin/bash

# Start the AI processor in the background
cd "$(dirname "$0")"

# Make sure we're using the stub version as we're just testing the HTTP interface
if [ ! -f "./bin/otel-ai-processor" ]; then
    echo "Building stub version for testing..."
    ./build-stub.sh
fi

echo "Starting the AI processor..."
./bin/otel-ai-processor --config=./config/config.yaml &
PROCESSOR_PID=$!

# Wait for processor to start
echo "Waiting for processor to start up..."
sleep 3

# Send test trace data
echo "Sending test trace data with otel-cli..."

# Test trace with error
echo "Sending error trace..."
otel-cli exec \
  --service "test-service" \
  --name "test-span-with-error" \
  --endpoint http://localhost:14318 \
  --attrs "operation.type=database,db.system=postgresql,error=true,message=Connection refused" \
  -- echo "Simulating operation with database error"

# Test trace with slow operation
echo "Sending slow operation trace..."
otel-cli exec \
  --service "test-service" \
  --name "test-span-slow-operation" \
  --endpoint http://localhost:14318 \
  --attrs "operation.type=http,http.method=GET,duration.ms=750" \
  -- sleep 0.75 && echo "Simulating slow HTTP operation"

# Test normal operation trace
echo "Sending normal operation trace..."
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

echo "Done testing with otel-cli!"