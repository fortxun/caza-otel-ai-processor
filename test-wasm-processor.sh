#\!/bin/bash

# Test script for AI-Enhanced Telemetry Processor (WASM version) using otel-cli
# This script will start the WASM processor and send test data to it

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
PROCESSOR_BINARY="./bin/otel-ai-processor-wasm"
CONFIG_FILE="./config/config.yaml"
LOG_FILE="processor-wasm.log"
PORT=4318

echo -e "${YELLOW}Testing AI-Enhanced Telemetry Processor (WASM version) with otel-cli${NC}"

# Check if the processor binary exists
if [ \! -f "$PROCESSOR_BINARY" ]; then
  echo -e "${RED}Error: Processor binary not found at $PROCESSOR_BINARY${NC}"
  exit 1
fi

# Kill any existing processor instances
pkill -f "otel-ai-processor" || true
sleep 1

# Start the processor
echo "Starting the WASM processor..."
nohup $PROCESSOR_BINARY --config=$CONFIG_FILE > $LOG_FILE 2>&1 &
PROCESSOR_PID=$\!
echo "Processor started with PID: $PROCESSOR_PID"

# Give the processor time to start up
echo "Waiting for processor to initialize..."
sleep 5

# Make sure the processor is running
if \! ps -p $PROCESSOR_PID > /dev/null; then
  echo -e "${RED}Error: Processor failed to start. Check $LOG_FILE for details.${NC}"
  exit 1
fi

echo -e "${GREEN}Processor is running. Starting tests...${NC}"

# Send a trace with error (should trigger error classification)
echo -e "\n${GREEN}Sending trace with error...${NC}"
otel-cli span send \
  --service "test-service" \
  --name "test.error.operation" \
  --status-code "ERROR" \
  --status-description "Test error for classification" \
  -a "test.severity=high" \
  -a "test.component=database" \
  --endpoint "http://localhost:$PORT"

# Send a slow span (should trigger sampling logic)
echo -e "\n${GREEN}Sending slow span...${NC}"
otel-cli span send \
  --service "test-service" \
  --name "test.slow.operation" \
  --status-code "OK" \
  -a "test.component=api" \
  -a "test.duration_ms=750" \
  --endpoint "http://localhost:$PORT"

# Send a normal span (for baseline)
echo -e "\n${GREEN}Sending normal span...${NC}"
otel-cli span send \
  --service "test-service" \
  --name "test.normal.operation" \
  --status-code "OK" \
  -a "test.component=web" \
  --endpoint "http://localhost:$PORT"

echo -e "\n${GREEN}All test data sent successfully\!${NC}"

# Display the most recent log output
echo -e "\n${YELLOW}Last 20 lines of processor log:${NC}"
tail -n 20 $LOG_FILE

# Keep the processor running for 5 more seconds
sleep 5

# Check for WASM-related log messages
echo -e "\n${YELLOW}Checking for WASM-related log messages:${NC}"
grep -i "wasm" $LOG_FILE

# Stop the processor
echo "Stopping processor..."
kill $PROCESSOR_PID
echo "Processor stopped."
