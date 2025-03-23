#\!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Set output directory
mkdir -p bin

echo "Building with fullwasm tag to test refactoring..."

# Just compile, don't build a binary
echo "Running 'go build -tags=fullwasm ./pkg/processor/...' to check for compilation errors"
CGO_ENABLED=0 go build -tags=fullwasm ./pkg/processor/...

# Check if build was successful
if [ $? -eq 0 ]; then
  echo ""
  echo "✅ Compilation with fullwasm tag completed successfully\!"
  echo ""
  echo "All refactored code compiles correctly with the fullwasm tag."
else
  echo ""
  echo "❌ Compilation failed."
  echo "Check the error messages above for more details."
fi
