# Docker Deployment

This guide covers deploying the AI-Enhanced Telemetry Processor using Docker.

## Prerequisites

- Docker 20.10.0 or later
- Docker Compose 2.0.0 or later (optional)
- Basic understanding of Docker concepts

## Using the Pre-built Docker Image

The simplest way to deploy the processor is to use the pre-built Docker image:

```bash
docker pull fortxun/caza-otel-ai-processor:latest
```

## Running with Docker

### Basic Docker Run Command

```bash
docker run -d \
  --name otel-ai-processor \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/config:/config \
  -v $(pwd)/models:/models \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

### With Environment Variables

You can override configuration settings using environment variables:

```bash
docker run -d \
  --name otel-ai-processor \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/config:/config \
  -v $(pwd)/models:/models \
  -e OTEL_PROCESSOR_AI_CONCURRENCY=8 \
  -e OTEL_PROCESSOR_AI_SAMPLING_NORMAL_SPANS=0.2 \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

### With Health Checks

Adding health checks to ensure the processor stays healthy:

```bash
docker run -d \
  --name otel-ai-processor \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 13133:13133 \
  -v $(pwd)/config:/config \
  -v $(pwd)/models:/models \
  --health-cmd "curl -f http://localhost:13133 || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

## Using Docker Compose

Docker Compose provides a more manageable way to deploy the processor, especially when combined with other services.

### Basic docker-compose.yml

```yaml
version: '3'
services:
  otel-collector:
    image: fortxun/caza-otel-ai-processor:latest
    container_name: otel-ai-processor
    command: ["--config=/config/config.yaml"]
    volumes:
      - ./config:/config
      - ./models:/models
    ports:
      - "4317:4317"
      - "4318:4318"
      - "13133:13133"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:13133"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Complete Example with Prometheus

Here's a more complete example that includes Prometheus for monitoring:

```yaml
version: '3'
services:
  otel-collector:
    image: fortxun/caza-otel-ai-processor:latest
    container_name: otel-ai-processor
    command: ["--config=/config/config.yaml"]
    volumes:
      - ./config:/config
      - ./models:/models
    ports:
      - "4317:4317"
      - "4318:4318"
      - "13133:13133"
      - "8888:8888"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:13133"]
      interval: 30s
      timeout: 10s
      retries: 3
    environment:
      - OTEL_PROCESSOR_AI_CONCURRENCY=4
    
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    depends_on:
      - otel-collector
```

## Building a Custom Docker Image

If you need to customize the Docker image:

1. Clone the repository:
   ```bash
   git clone https://github.com/fortxun/caza-otel-ai-processor.git
   cd caza-otel-ai-processor
   ```

2. Modify the Dockerfile or build scripts as needed

3. Build the image:
   ```bash
   docker build -t custom-otel-ai-processor:latest .
   ```

## Resource Constraints

Specify resource limits to ensure the processor doesn't consume too many resources:

```bash
docker run -d \
  --name otel-ai-processor \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/config:/config \
  -v $(pwd)/models:/models \
  --cpus=2 \
  --memory=512m \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

## Persistent Storage

If you need to store processor data persistently:

```bash
docker run -d \
  --name otel-ai-processor \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/config:/config \
  -v $(pwd)/models:/models \
  -v otel-data:/var/lib/otel \
  fortxun/caza-otel-ai-processor:latest \
  --config=/config/config.yaml
```

## Docker Network Configuration

If you're running multiple containers that need to communicate:

```yaml
version: '3'
services:
  otel-collector:
    image: fortxun/caza-otel-ai-processor:latest
    networks:
      - otel-net
    # other configuration...
  
  app:
    image: your-app:latest
    networks:
      - otel-net
    # other configuration...

networks:
  otel-net:
    driver: bridge
```

## Next Steps

- Learn about [Kubernetes deployment](./kubernetes.md)
- Explore [performance tuning](../performance/tuning.md)
- Set up [monitoring](../troubleshooting/logs-metrics.md)