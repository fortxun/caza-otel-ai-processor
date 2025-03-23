# AI-Enhanced Telemetry Processor Tests

This directory contains tests for the AI-Enhanced Telemetry Processor.

## Test Structure

- **Unit Tests**: Tests for individual components
  - `traces_test.go`: Tests for the trace processor
  - `metrics_test.go`: Tests for the metrics processor
  - `logs_test.go`: Tests for the logs processor
  - `config_test.go`: Tests for configuration validation

- **Integration Tests**: Tests for the entire processor pipeline
  - `integration_test.go`: Tests integration of components

- **Benchmarks**: Performance tests
  - `benchmark_test.go`: Performance benchmarks for critical paths

- **Utilities**:
  - `mocks.go`: Mock implementations for testing

## Running Tests

### Run all tests:

```bash
go test ./pkg/... -v
```

### Run unit tests only:

```bash
go test ./pkg/... -run "^Test" -v
```

### Run integration tests only:

```bash
go test ./pkg/... -run "^TestProcessor" -v
```

### Run benchmarks:

```bash
go test ./pkg/... -bench=. -benchmem
```

### Short Tests (skipping integration tests):

```bash
go test ./pkg/... -short -v
```

## Test Coverage

Generate test coverage report:

```bash
go test ./pkg/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Mock Functions

The `mocks.go` file provides mock implementations for testing:

- `MockWasmRuntime`: Mock WASM runtime with predetermined responses
- `MockTracesConsumer`: Mock traces consumer
- `MockMetricsConsumer`: Mock metrics consumer
- `MockLogsConsumer`: Mock logs consumer
- `TestData`: Test data generators for traces, metrics, and logs

## Benchmarking Guidelines

When making changes to the codebase, run the benchmarks to ensure performance remains within requirements:

- Processor latency should be under 10ms per batch
- Memory usage should be minimal
- CPU utilization should be efficient

The most important benchmark results are:

- `BenchmarkTracesProcessor_WithFeatures`: Tests processing traces with all features enabled
- `BenchmarkSamplingDecision`: Tests the sampling decision logic
- `BenchmarkErrorClassification`: Tests the error classification logic

## Integration Test Notes

The integration tests verify end-to-end functionality including:

1. Factory creation and configuration
2. Processor instantiation
3. Telemetry processing (traces, metrics, logs)
4. WASM model interaction
5. Proper shutdown

In CI environments, the integration tests are skipped with the `-short` flag since they require WASM model files. For local development, you can build the WASM models following instructions in the `wasm-models` directory.