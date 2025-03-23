#!/bin/bash

# Build and deploy WASM models for OpenTelemetry AI Processor

# Ensure script is run from the wasm-models directory
cd "$(dirname "$0")"

echo "Installing dependencies..."
npm install

echo "Building WASM models..."
npm run asbuild

echo "Creating model directory if it doesn't exist..."
mkdir -p ../models

echo "Copying WASM files to models directory..."
cp error-classifier/build/error-classifier.wasm ../models/
cp importance-sampler/build/importance-sampler.wasm ../models/
cp entity-extractor/build/entity-extractor.wasm ../models/

echo "WASM models built and deployed successfully!"
echo "Model sizes:"
ls -lh ../models/*.wasm | awk '{print $9, $5}'

echo "Done!"