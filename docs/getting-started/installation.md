# Installation Guide

This guide covers the different ways to install the AI-Enhanced Telemetry Processor for OpenTelemetry.

## Option 1: Building from Source

### Prerequisites

- Go 1.20 or later
- Git
- Make (optional, but recommended)

### Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/fortxun/caza-otel-ai-processor.git
   cd caza-otel-ai-processor
   ```

2. Build the processor:
   ```bash
   go build -o otel-ai-processor ./cmd/processor
   ```

   Alternatively, use the provided Makefile:
   ```bash
   make build
   ```

3. Build the WASM models:
   ```bash
   cd wasm-models
   npm install
   npm run asbuild
   ```

   Or use the convenience script:
   ```bash
   ./build-models.sh
   ```

4. Copy the compiled WASM models to the models directory:
   ```bash
   mkdir -p ../models
   cp error-classifier/build/error-classifier.wasm ../models/
   cp importance-sampler/build/importance-sampler.wasm ../models/
   cp entity-extractor/build/entity-extractor.wasm ../models/
   ```

## Option 2: Using Docker

### Prerequisites

- Docker
- Docker Compose (optional, for using the provided docker-compose.yml)

### Steps

1. Pull the pre-built image:
   ```bash
   docker pull fortxun/caza-otel-ai-processor:latest
   ```

   Or build the image locally:
   ```bash
   docker build -t fortxun/caza-otel-ai-processor:latest .
   ```

2. Run the processor using Docker:
   ```bash
   docker run -p 4317:4317 -p 4318:4318 -v $(pwd)/config:/config fortxun/caza-otel-ai-processor:latest --config=/config/config.yaml
   ```

3. Alternatively, use Docker Compose:
   ```bash
   docker-compose up -d
   ```

## Option 3: OpenTelemetry Collector Builder

If you're using the OpenTelemetry Collector Builder to create a custom distribution, you can include the AI-Enhanced Telemetry Processor in your build.

### Steps

1. Add the processor to your builder configuration (builder-config.yaml):
   ```yaml
   processors:
     - gomod: github.com/fortxun/caza-otel-ai-processor/cmd/processor v0.1.0
   ```

2. Build your custom collector:
   ```bash
   builder --config=builder-config.yaml
   ```

## Verifying the Installation

After installation, you can verify that the processor is working correctly by:

1. Starting the OpenTelemetry Collector with the processor configured
2. Sending some test telemetry through the collector
3. Checking that the telemetry is processed and enriched correctly

See the [Quick Start Guide](./quick-start.md) for more details on these steps.

## Next Steps

Once the processor is installed, the next step is to configure it for your specific needs:

- [Basic Configuration](./basic-configuration.md)
- [Quick Start Guide](./quick-start.md)
- [Advanced Configuration](../configuration/index.md)