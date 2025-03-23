# Performance Tuning Guidelines

This guide provides detailed information on optimizing the performance of the AI-Enhanced Telemetry Processor.

## Understanding Performance Factors

The performance of the processor is affected by several factors:

1. **WASM Model Execution**: The time it takes to execute the models
2. **Data Volume**: The amount of telemetry being processed
3. **Resource Availability**: CPU and memory constraints
4. **Configuration Settings**: Batch sizes, concurrency, caching options

## Key Performance Parameters

### Processing Configuration

The `processing` section in the configuration controls the most important performance parameters:

```yaml
processing:
  batch_size: 50       # Number of items to process in a batch
  concurrency: 4       # Number of parallel workers
  queue_size: 1000     # Size of the processing queue
  timeout_ms: 500      # Overall processing timeout
```

#### Batch Size

The `batch_size` parameter controls how many telemetry items are processed together in a batch:

- **Smaller batches**: Faster individual batch processing, but higher overhead
- **Larger batches**: More efficient overall processing, but higher latency
- **Recommended range**: 10-100, depending on telemetry complexity

#### Concurrency

The `concurrency` parameter determines how many worker goroutines are created to process telemetry:

- **Lower concurrency**: Less resource usage, but potentially slower processing
- **Higher concurrency**: Faster processing, but higher resource usage
- **Recommended setting**: Start with the number of available CPU cores, then adjust as needed

#### Queue Size

The `queue_size` parameter controls the size of the internal processing queue:

- **Smaller queue**: Less memory usage, but more likely to block or drop telemetry under load
- **Larger queue**: More resilient to bursts, but higher memory usage
- **Recommended range**: 500-5000, depending on expected telemetry volume

### WASM Model Configuration

Each model has configuration parameters that affect performance:

```yaml
models:
  error_classifier:
    path: "/models/error-classifier.wasm"
    memory_limit_mb: 100
    timeout_ms: 50
    cache_size: 1000
```

#### Memory Limit

The `memory_limit_mb` parameter restricts how much memory each WASM instance can use:

- **Lower limit**: Less memory usage, but might restrict model capabilities
- **Higher limit**: More flexibility for models, but higher memory usage
- **Recommended ranges**:
  - Error classifier: 50-150 MB
  - Importance sampler: 30-100 MB
  - Entity extractor: 100-200 MB

#### Execution Timeout

The `timeout_ms` parameter limits how long a model can run:

- **Lower timeout**: Faster pipeline, but might truncate complex processing
- **Higher timeout**: More reliable processing, but might slow down the pipeline
- **Recommended ranges**:
  - Error classifier: 30-100 ms
  - Importance sampler: 20-50 ms
  - Entity extractor: 30-100 ms

#### Cache Size

The `cache_size` parameter determines how many model results are cached:

- **Smaller cache**: Less memory usage, but more repeated computations
- **Larger cache**: Faster processing for repeated patterns, but higher memory usage
- **Recommended range**: 500-5000, depending on pattern diversity in your telemetry

## Optimizing for Different Scenarios

### High-Volume Environment

For environments with high telemetry volume:

```yaml
processing:
  batch_size: 100
  concurrency: 8
  queue_size: 5000
  timeout_ms: 1000

models:
  error_classifier:
    memory_limit_mb: 150
    timeout_ms: 75
    cache_size: 5000
  # Similar adjustments for other models
```

### Resource-Constrained Environment

For environments with limited resources:

```yaml
processing:
  batch_size: 20
  concurrency: 2
  queue_size: 500
  timeout_ms: 300

models:
  error_classifier:
    memory_limit_mb: 50
    timeout_ms: 30
    cache_size: 500
  # Similar adjustments for other models
```

### Low-Latency Environment

For environments where processing latency is critical:

```yaml
processing:
  batch_size: 10
  concurrency: 16
  queue_size: 1000
  timeout_ms: 200

models:
  error_classifier:
    memory_limit_mb: 100
    timeout_ms: 25
    cache_size: 2000
  # Similar adjustments for other models
```

## Advanced Tuning Techniques

### Selective Processing

Configure feature toggles to process only what you need:

```yaml
features:
  error_classification: true  # Only enable needed features
  smart_sampling: true
  entity_extraction: false    # Disable features you don't need
```

### Sampling Strategies

Adjust sampling rates based on telemetry importance:

```yaml
sampling:
  error_events: 1.0     # Keep all errors
  slow_spans: 0.8       # Keep most slow spans
  normal_spans: 0.05    # Keep very few normal spans
```

### Resource Allocation

Ensure the processor has adequate resources:

- **CPU**: Ideally, at least as many cores as your concurrency setting
- **Memory**: At least 256MB plus the sum of all model memory limits
- **Scheduling**: Use CPU affinity settings to dedicate cores to the processor

### Operating System Tuning

For Linux systems:

- Increase file descriptor limits (`ulimit -n`)
- Optimize CPU scaling governor for performance
- Adjust network buffer sizes for high-throughput scenarios

## Performance Monitoring

Monitor key metrics to identify bottlenecks:

- **CPU usage**: Per worker and overall
- **Memory usage**: Overall and per WASM instance
- **Queue depth**: How full the processing queue gets
- **Processing latency**: Time from ingestion to completion
- **Cache hit ratio**: Effectiveness of result caching

## Benchmarking

Use the included benchmarks to test your configuration:

```bash
make test-bench
```

Or create custom load tests to simulate your expected traffic patterns.

## Troubleshooting Performance Issues

Common performance issues and solutions:

1. **High CPU usage**:
   - Reduce concurrency
   - Simplify models
   - Increase batch size

2. **High memory usage**:
   - Reduce model memory limits
   - Decrease cache sizes
   - Lower queue size

3. **Processing delays**:
   - Increase concurrency
   - Optimize models
   - Consider horizontal scaling

4. **Dropped telemetry**:
   - Increase queue size
   - Adjust batch size
   - Implement back-pressure mechanisms