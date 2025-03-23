# Configuration Guide

This section provides detailed information about configuring the AI-Enhanced Telemetry Processor for OpenTelemetry.

## Configuration Format

The processor configuration follows the standard OpenTelemetry Collector configuration format, using YAML. The processor section is identified by the `ai_processor` key.

## Configuration Sections

The configuration is organized into several sections:

- [Processor Configuration](./processor.md): General processor settings
- [Model Configuration](./models.md): WASM model settings
- [Performance Settings](./performance.md): Tuning for optimal performance
- [Feature Toggles](./features.md): Enabling or disabling specific functionality
- [Sampling Configuration](./sampling.md): Smart sampling behavior
- [Output Settings](./output.md): Controlling how processed data is exported

## Complete Configuration Example

Here's a complete configuration example showing all available options:

```yaml
processors:
  ai_processor:
    # Models configuration
    models:
      error_classifier:
        path: "/models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
        cache_size: 1000
      importance_sampler:
        path: "/models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30
        cache_size: 1000
      entity_extractor:
        path: "/models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50
        cache_size: 1000

    # Processing settings
    processing:
      batch_size: 50
      concurrency: 4
      queue_size: 1000
      timeout_ms: 500
      retry_count: 3
      retry_delay_ms: 100
      buffer_size: 2000

    # Feature toggles
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: true
      context_linking: false
      attribute_caching: true
      resource_caching: true
      model_result_caching: true
      debug_logging: false

    # Sampling configuration
    sampling:
      error_events: 1.0  # Keep all errors
      slow_spans: 1.0    # Keep all slow spans
      normal_spans: 0.1  # Keep 10% of normal spans
      threshold_ms: 500  # Slow span threshold
      min_duration_ms: 10  # Minimum duration to consider for sampling
      importance_threshold: 0.5  # Importance score threshold
      random_sampling_seed: 42  # Seed for random sampling (optional)

    # Output configuration
    output:
      attribute_namespace: "ai."
      include_confidence_scores: true
      max_attribute_length: 256
      truncate_strings: true
      flatten_arrays: false
      merge_behavior: "replace"  # "replace", "merge", or "preserve"
      debug_attributes: false
```

## Environment Variable Overrides

Configuration settings can also be specified using environment variables, using the following format:

```
OTEL_PROCESSOR_AI_<SECTION>_<OPTION>=<value>
```

For example, to set the concurrency to 8:

```bash
export OTEL_PROCESSOR_AI_PROCESSING_CONCURRENCY=8
```

To enable debug logging:

```bash
export OTEL_PROCESSOR_AI_FEATURES_DEBUG_LOGGING=true
```

## Configuration Validation

The processor validates the configuration at startup, checking for:

- Required fields
- Value constraints
- File existence
- Incompatible settings

If there are any validation errors, the processor will log detailed error messages and fail to start.

## Next Steps

- [Getting Started Guide](../getting-started/index.md)
- [Performance Tuning](../performance/tuning.md)
- [Example Configurations](../examples/index.md)