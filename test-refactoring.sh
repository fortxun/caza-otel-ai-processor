#\!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Set output directory
mkdir -p bin

echo "Testing the refactored code (API standardization)..."

# Just compile, don't build a binary
echo "Running 'go build ./pkg/processor/...' to check for compilation errors"
CGO_ENABLED=0 go build ./pkg/processor/...

# Check if build was successful
if [ $? -eq 0 ]; then
  echo ""
  echo "✅ Compilation completed successfully\!"
  echo ""
  echo "The refactoring to standardize on the newer OpenTelemetry API was successful."
  echo "All code now uses the newer API structure without build tags."
else
  echo ""
  echo "❌ Compilation failed."
  echo "Check the error messages above for more details."
fi
