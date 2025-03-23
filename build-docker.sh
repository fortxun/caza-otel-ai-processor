#!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
  echo "Error: Docker is not installed or not in PATH"
  echo "Please install Docker before continuing:"
  echo "  - Mac: https://docs.docker.com/desktop/install/mac-install/"
  echo "  - Linux: https://docs.docker.com/engine/install/"
  echo "  - Windows: https://docs.docker.com/desktop/install/windows-install/"
  exit 1
fi

echo "Building Docker image: fortxun/caza-otel-ai-processor:latest"
echo "This may take a few minutes..."

# Build the Docker image
docker build -t fortxun/caza-otel-ai-processor:latest .

# Check if the build was successful
if [ $? -eq 0 ]; then
  echo "✅ Docker build completed successfully!"
  echo ""
  echo "To run the container with the models, use:"
  echo "docker run -p 4317:4317 -p 4318:4318 -v $(pwd)/config:/config fortxun/caza-otel-ai-processor:latest --config=/config/config.yaml"
  echo ""
  echo "To push the image to Docker Hub (after logging in):"
  echo "docker push fortxun/caza-otel-ai-processor:latest"
else
  echo "❌ Docker build failed."
  echo "Check the build logs above for errors."
  echo "For more detailed troubleshooting, see the BUILD_GUIDE.md file."
fi