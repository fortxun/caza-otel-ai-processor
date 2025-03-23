#!/bin/bash

# Verification script for AI-Enhanced Telemetry Processor
# This script checks if the processor is running and listening on expected ports

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}AI-Enhanced Telemetry Processor Status Check${NC}"

# Check if processor binary exists
if [ ! -f "./bin/otel-ai-processor" ]; then
  echo -e "${RED}ERROR: Processor binary not found${NC}"
  echo "Run './build-stub.sh' to build the processor first"
  exit 1
fi

# Check if processor is running
PROCESSOR_PID=$(pgrep -f "otel-ai-processor")
if [ -z "$PROCESSOR_PID" ]; then
  echo -e "${RED}Processor is not running${NC}"
  echo "Start it with: ./bin/otel-ai-processor"
else
  echo -e "${GREEN}Processor is running with PID: $PROCESSOR_PID${NC}"
fi

# Check listening ports
echo -e "\n${YELLOW}Checking listening ports:${NC}"

# Check OTLP gRPC port
if netstat -tuln 2>/dev/null | grep -q ":4317 "; then
  echo -e "${GREEN}✓ OTLP gRPC port (4317) is open and listening${NC}"
else
  echo -e "${RED}✗ OTLP gRPC port (4317) is not listening${NC}"
fi

# Check OTLP HTTP port
if netstat -tuln 2>/dev/null | grep -q ":4318 "; then
  echo -e "${GREEN}✓ OTLP HTTP port (4318) is open and listening${NC}"
else
  echo -e "${RED}✗ OTLP HTTP port (4318) is not listening${NC}"
fi

# Verify processor stub configuration
echo -e "\n${YELLOW}Checking stub implementation:${NC}"
if grep -q "fullwasm" "$(which ./bin/otel-ai-processor)" 2>/dev/null; then
  echo -e "${RED}WARNING: Processor was built with fullwasm tag${NC}"
  echo "This means it will try to use actual WASM models, not stubs"
else
  echo -e "${GREEN}✓ Processor is using stub implementation (no fullwasm tag)${NC}"
fi

# Report processor memory usage
if [ ! -z "$PROCESSOR_PID" ]; then
  MEM_USAGE=$(ps -o rss= -p $PROCESSOR_PID | awk '{print $1/1024}')
  echo -e "\n${YELLOW}Processor memory usage: ${GREEN}${MEM_USAGE} MB${NC}"
fi

echo -e "\n${YELLOW}To test the processor:${NC}"
echo "1. Make sure the processor is running"
echo "2. Run './test-processor.sh' to send sample telemetry"
echo "3. Run './test-model-stubs.sh' to test WASM model stub functions"
echo "4. Check processor logs with: tail -f processor.log"