#!/bin/bash

# Test script specifically for the WASM model stub implementations
# This will explicitly test each of the model functions

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing AI-Enhanced Telemetry Processor WASM Model Stubs${NC}"

# Make sure the processor is running for log observation
if ! pgrep -f "otel-ai-processor" > /dev/null; then
  echo "Starting the processor..."
  nohup ./bin/otel-ai-processor > processor.log 2>&1 &
  PROCESSOR_PID=$!
  echo "Processor started with PID: $PROCESSOR_PID"
  # Give the processor a moment to start up
  sleep 2
else
  echo "Processor is already running"
fi

# Clear the log file for fresh test results
> processor.log

echo -e "\n${BLUE}1. Testing Error Classifier Model${NC}"
echo "Sending trace with error to trigger error classification..."
otel-cli span send \
  --service "test-service" \
  --name "error.classification.test" \
  --status-code error \
  --status-description "Database connection timeout" \
  -a "db.system=postgres" \
  -a "error.message=Connection refused" \
  --endpoint "http://localhost:4318"

# Wait a moment for processing
sleep 1
echo -e "${GREEN}Checking logs for Error Classifier output:${NC}"
grep -E "Status code.*ERROR|Stub ClassifyError called" processor.log

echo -e "\n${BLUE}2. Testing Importance Sampler Model${NC}"
echo "Sending span to trigger importance sampling..."
otel-cli span send \
  --service "test-service" \
  --name "db.query" \
  --status-code "OK" \
  -a "db.statement=SELECT * FROM users" \
  -a "db.system=mysql" \
  --endpoint "http://localhost:4318"

# Wait a moment for processing
sleep 1
echo -e "${GREEN}Checking logs for Importance Sampler output:${NC}"
grep "Stub SampleTelemetry called" processor.log

echo -e "\n${BLUE}3. Testing Entity Extractor Model${NC}"
echo "Sending span to trigger entity extraction..."
otel-cli span send \
  --service "test-service" \
  --name "http.request" \
  --status-code "OK" \
  -a "http.method=GET" \
  -a "http.url=https://api.example.com/users/123" \
  -a "http.status_code=200" \
  --endpoint "http://localhost:4318"

# Wait a moment for processing
sleep 1
echo -e "${GREEN}Checking logs for Entity Extractor output:${NC}"
grep "Stub ExtractEntities called" processor.log

echo -e "\n${YELLOW}Test Summary${NC}"
echo "Error Classification test: $(grep -q 'Stub ClassifyError called' processor.log && echo 'PASSED' || echo 'FAILED')"
echo "Importance Sampling test: $(grep -q 'Stub SampleTelemetry called' processor.log && echo 'PASSED' || echo 'FAILED')"
echo "Entity Extraction test: $(grep -q 'Stub ExtractEntities called' processor.log && echo 'PASSED' || echo 'FAILED')"

echo -e "\n${GREEN}Testing complete!${NC}"
echo "For detailed results, check: processor.log"