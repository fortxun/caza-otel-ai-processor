# AI-Enhanced Telemetry Processor for OpenTelemetry

A lightweight, CPU-only AI processor for OpenTelemetry that enhances telemetry data through intelligent classification, contextual enrichment, and smart sampling. This processor integrates seamlessly with existing OpenTelemetry deployments without requiring changes to instrumented applications, using WebAssembly (WASM) models for local execution.

## Key Features

- **Error Classification**: Automatically categorize errors, identify affected systems, and suggest owners
- **Smart Sampling**: Reduce data volume while retaining important telemetry through content-aware sampling
- **Entity Extraction**: Identify services, dependencies, and operations from telemetry data
- **WASM Model Integration**: Efficient, isolated execution of AI models with low resource footprint
- **Parallel Processing**: Concurrent telemetry processing for high throughput
- **Result Caching**: Optimized performance through caching of model results

## Benefits

- Reduce storage costs by 30-50% through intelligent sampling
- Improve incident response time with automatic error classification
- Enhance signal-to-noise ratio in telemetry data
- No changes required to application instrumentation
- Minimal CPU and memory footprint

## System Requirements

- CPU-only operation (no GPU required)
- Maximum 500MB memory footprint
- CPU usage below 2 cores at peak
- Works with standard OTLP data formats

## Documentation

Comprehensive documentation is available in the `/docs` directory:

- [Getting Started](./docs/getting-started/index.md)
  - [Installation](./docs/getting-started/installation.md)
  - [Quick Start Guide](./docs/getting-started/quick-start.md)
  - [Basic Configuration](./docs/getting-started/basic-configuration.md)

- [Configuration](./docs/configuration/index.md)
- [Deployment](./docs/deployment/index.md)
- [Performance Tuning](./docs/performance/tuning.md)
- [Troubleshooting](./docs/troubleshooting/common-issues.md)
- [API Reference](./docs/api-reference/processor.md)
- [Example Integrations](./docs/examples/integration.md)
- [Model Customization](./docs/examples/model-customization.md)

## Quick Start

### Prerequisites

- Go 1.23+ (required for latest OpenTelemetry dependencies)
- OpenTelemetry Collector v0.95.0+
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/fortxun/caza-otel-ai-processor.git
   ```

2. Build the processor using one of these methods:

   **Using the convenience script (recommended):**
   ```bash
   # Local build script (no Docker required)
   ./build-local.sh
   
   # Docker build script (if Docker is installed)
   ./build-docker.sh
   ```

   **Manual build:**
   ```bash
   # Local build
   go mod download && go mod tidy
   go build -o ./bin/otel-ai-processor ./cmd/processor
   
   # Docker build
   docker build -t fortxun/caza-otel-ai-processor:latest .
   ```

3. For detailed build instructions and troubleshooting, see [Build Guide](./docs/BUILD_GUIDE.md)

### Running Tests

The project includes comprehensive tests to ensure functionality and performance:

```bash
# Run all tests
make test

# Run tests with coverage report
make test-cover

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make test-bench
```

### CI/CD Pipeline

This project uses GitHub Actions for continuous integration and testing:

- **Stub Implementation Testing**: Builds and tests the processor with the stub WASM implementation
- **Full WASM Implementation Testing**: Builds the WASM models and tests the processor with the full WASM implementation
- **Performance Benchmarking**: Runs performance comparisons between the stub and WASM implementations
- **Docker Builds**: Creates Docker images for both implementations

The CI pipeline runs automatically on pull requests and pushes to the main branch, ensuring both implementations remain functional.

To run the workflow manually, go to the Actions tab in the GitHub repository and select "CI Pipeline" from the workflows list.

[![CI Pipeline Status](https://github.com/fortxun/caza-otel-ai-processor/actions/workflows/ci.yml/badge.svg)](https://github.com/fortxun/caza-otel-ai-processor/actions/workflows/ci.yml)

### Building WASM Models

The AI models need to be built and placed in the `/models` directory:

```bash
cd wasm-models
./build-models.sh
```

### Basic Configuration

Add the processor to your OpenTelemetry Collector configuration:

```yaml
processors:
  ai_processor:
    # Models configuration
    models:
      error_classifier:
        path: "/models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
      importance_sampler:
        path: "/models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30
      entity_extractor:
        path: "/models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50

    # Processing settings
    processing:
      batch_size: 50
      concurrency: 4
      queue_size: 1000
      timeout_ms: 500

    # Feature toggles
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: true

    # Sampling configuration
    sampling:
      error_events: 1.0  # Keep all errors
      slow_spans: 1.0    # Keep all slow spans
      normal_spans: 0.1  # Keep 10% of normal spans
      threshold_ms: 500  # Slow span threshold

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [otlp]
    metrics:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [otlp]
    logs:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [otlp]
```

For detailed configuration options, see the [Configuration Guide](./docs/configuration/index.md).

## Project Structure

```
├── cmd/
│   └── processor/        # Main application entry point
├── pkg/
│   ├── processor/        # OpenTelemetry processor implementation
│   ├── runtime/          # WASM runtime integration
│   └── config/           # Configuration handling
├── models/               # Pre-built WASM models
├── wasm-models/          # WASM model source code
│   ├── error-classifier/ # Error classifier model
│   ├── importance-sampler/ # Sampling model
│   └── entity-extractor/ # Entity extraction model
├── config/               # Configuration examples
└── docs/                 # Documentation
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.