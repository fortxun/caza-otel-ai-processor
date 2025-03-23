# Basic Configuration

This guide covers the essential configuration options for the AI-Enhanced Telemetry Processor.

## Configuration File Structure

The processor uses YAML for configuration, following the OpenTelemetry Collector configuration format. Here's a basic structure:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  ai_processor:
    # Configuration options go here

exporters:
  otlp:
    endpoint: "localhost:4319"
    tls:
      insecure: true

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

## Essential Configuration Options

### 1. Model Configuration

The first step is to configure the WASM models:

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
```

Each model requires:
- `path`: Location of the WASM file
- `memory_limit_mb`: Maximum memory allocation for the WASM instance
- `timeout_ms`: Maximum execution time in milliseconds

### 2. Processing Settings

Configure performance-related options:

```yaml
processors:
  ai_processor:
    # Processing settings
    processing:
      batch_size: 50        # Number of items to process in a batch
      concurrency: 4        # Number of parallel workers
      queue_size: 1000      # Size of the processing queue
      timeout_ms: 500       # Overall processing timeout
```

### 3. Feature Toggles

Enable or disable specific functionality:

```yaml
processors:
  ai_processor:
    # Feature toggles
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: true
      context_linking: false
```

### 4. Sampling Configuration

Configure sampling behavior:

```yaml
processors:
  ai_processor:
    # Sampling configuration
    sampling:
      error_events: 1.0     # Keep all errors
      slow_spans: 1.0       # Keep all slow spans
      normal_spans: 0.1     # Keep 10% of normal spans
      threshold_ms: 500     # Slow span threshold
```

### 5. Output Configuration

Configure how processed data is output:

```yaml
processors:
  ai_processor:
    # Output configuration
    output:
      attribute_namespace: "ai."
      include_confidence_scores: true
      max_attribute_length: 256
```

## Minimal Configuration Example

Here's a minimal configuration to get started:

```yaml
processors:
  ai_processor:
    models:
      error_classifier:
        path: "/models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
      importance_sampler:
        path: "/models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30

    processing:
      concurrency: 4
      
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: false

    sampling:
      error_events: 1.0
      normal_spans: 0.1
```

## Configuration Validation

The processor validates the configuration at startup. If there are any issues, it will log detailed error messages and fail to start.

Common validation errors include:
- Missing or invalid model paths
- Invalid memory limits or timeouts
- Conflicting feature configurations

## Next Steps

For more detailed configuration options, see:

- [Advanced Configuration](../configuration/index.md)
- [Model Configuration](../configuration/models.md)
- [Performance Settings](../configuration/performance.md)