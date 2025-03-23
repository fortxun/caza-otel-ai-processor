# Quick Start Guide

This guide will help you get the AI-Enhanced Telemetry Processor up and running quickly.

## Prerequisites

- OpenTelemetry Collector installed
- Docker installed (optional, for containerized setup)

## Option 1: Using Docker Compose

The quickest way to start experimenting with the processor is to use the provided Docker Compose configuration.

1. Create a directory for your project:
   ```bash
   mkdir otel-ai-processor-demo
   cd otel-ai-processor-demo
   ```

2. Download the sample Docker Compose file:
   ```bash
   curl -O https://raw.githubusercontent.com/fortxun/caza-otel-ai-processor/main/docker-compose.yml
   curl -O https://raw.githubusercontent.com/fortxun/caza-otel-ai-processor/main/config/config.yaml
   ```

3. Start the services:
   ```bash
   docker-compose up -d
   ```

4. Check that the services are running:
   ```bash
   docker-compose ps
   ```

## Option 2: Using a Pre-built Binary

If you prefer to run the processor directly on your host machine:

1. Download the latest release:
   ```bash
   curl -L https://github.com/fortxun/caza-otel-ai-processor/releases/latest/download/otel-ai-processor-$(uname -s)-$(uname -m) -o otel-ai-processor
   chmod +x otel-ai-processor
   ```

2. Download the sample configuration:
   ```bash
   curl -O https://raw.githubusercontent.com/fortxun/caza-otel-ai-processor/main/config/config.yaml
   ```

3. Download the WASM models:
   ```bash
   mkdir -p models
   curl -L https://github.com/fortxun/caza-otel-ai-processor/releases/latest/download/models.tar.gz | tar -xz -C models
   ```

4. Start the processor:
   ```bash
   ./otel-ai-processor --config=config.yaml
   ```

## Sending Test Data

Now that the processor is running, you can send some test telemetry data to see it in action:

1. Use the OpenTelemetry Testing Client to generate traces:
   ```bash
   docker run --network=host otel/opentelemetry-collector-contrib:latest telemetrygen traces --url=http://localhost:4318 --rate=10
   ```

2. Generate logs:
   ```bash
   docker run --network=host otel/opentelemetry-collector-contrib:latest telemetrygen logs --url=http://localhost:4318 --rate=10
   ```

3. Generate metrics:
   ```bash
   docker run --network=host otel/opentelemetry-collector-contrib:latest telemetrygen metrics --url=http://localhost:4318 --rate=10
   ```

## Verifying the Results

You can verify that the processor is working correctly by:

1. Checking the collector logs:
   ```bash
   docker-compose logs -f otel-collector
   ```

   Or if running directly:
   ```bash
   tail -f otel-ai-processor.log
   ```

2. Looking for enriched telemetry data with attributes added by the AI models. The attributes will have the prefix specified in your configuration (default is "ai.").

## Sample Output

Here's an example of what enriched span data might look like:

```json
{
  "name": "GET /api/users",
  "trace_id": "0123456789abcdef0123456789abcdef",
  "span_id": "0123456789abcdef",
  "attributes": {
    "http.method": "GET",
    "http.url": "https://api.example.com/users",
    "ai.entity.services": ["user-service", "auth-service"],
    "ai.entity.dependencies": ["postgres", "redis"],
    "ai.entity.operations": ["fetch_users", "authenticate"],
    "ai.entity.confidence": 0.88
  }
}
```

## Next Steps

Now that you have the processor up and running, you can:

1. Customize the [configuration](./basic-configuration.md) to better fit your needs
2. Explore the [advanced configuration options](../configuration/index.md)
3. Learn about [deployment strategies](../deployment/index.md) for production environments