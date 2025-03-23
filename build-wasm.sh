#!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

echo "🧠 Building WASM models for CAZA OpenTelemetry AI Processor..."

# Check if node and npm are installed (required for AssemblyScript)
if ! command -v node &> /dev/null; then
  echo "❌ Error: Node.js is not installed. Please install Node.js (required for AssemblyScript)."
  exit 1
fi

if ! command -v npm &> /dev/null; then
  echo "❌ Error: npm is not installed. Please install npm (required for AssemblyScript)."
  exit 1
fi

# Create models directory if it doesn't exist
mkdir -p models

# Navigate to the wasm-models directory
cd wasm-models || { echo "❌ Error: wasm-models directory not found"; exit 1; }

# Install dependencies if not already installed
if [ ! -d "node_modules" ]; then
  echo "Installing AssemblyScript dependencies..."
  npm install || { echo "❌ Failed to install dependencies"; exit 1; }
fi

# Build the models
echo "Building error classifier model..."
cd error-classifier || { echo "❌ Error accessing error-classifier directory"; exit 1; }
npm run asbuild || { echo "❌ Failed to build error classifier model"; exit 1; }
cp build/error-classifier.wasm ../../models/
cd ..

echo "Building importance sampler model..."
cd importance-sampler || { echo "❌ Error accessing importance-sampler directory"; exit 1; }
npm run asbuild || { echo "❌ Failed to build importance sampler model"; exit 1; }
cp build/importance-sampler.wasm ../../models/
cd ..

echo "Building entity extractor model..."
cd entity-extractor || { echo "❌ Error accessing entity-extractor directory"; exit 1; }
npm run asbuild || { echo "❌ Failed to build entity extractor model"; exit 1; }
cp build/entity-extractor.wasm ../../models/
cd ..

echo ""
echo "✅ Successfully built and copied all WASM models to the models directory!"
echo ""
echo "Models built:"
echo "  - models/error-classifier.wasm"
echo "  - models/importance-sampler.wasm"
echo "  - models/entity-extractor.wasm"