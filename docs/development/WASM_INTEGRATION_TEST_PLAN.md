# WASM Integration Testing Implementation Plan

This document outlines the comprehensive integration testing implementation for the CAZA OpenTelemetry AI Processor with WASM models.

## Overview

The integration testing approach addresses the need to verify that both the stub and full WASM implementations function correctly. We've implemented a full set of tests and tools to ensure proper WASM model loading, execution, and performance analysis.

## Components Implemented

### 1. Integration Tests

Created a dedicated integration test file in `pkg/processor/tests/wasm_integration_test.go` that:

- Loads actual WASM models from the wasm-models directory
- Tests all three main AI functions:
  - Error classification
  - Telemetry sampling
  - Entity extraction
- Validates all three signal types:
  - Traces
  - Metrics
  - Logs
- Tests parallel processing with large batches of spans
- Verifies proper attribute annotation with AI results

### 2. Performance Benchmarks

Created comprehensive benchmarks in `pkg/processor/tests/wasm_benchmark_test.go` that:

- Compare the WASM implementation vs the stub implementation
- Benchmark each AI function individually
- Benchmark the full processing pipeline
- Measure CPU, memory usage, and execution time
- Use realistic test data that mimics production workloads

### 3. Test Scripts

#### WASM Integration Test Script

Created `test-fullwasm-integration.sh` that:
- Builds the WASM models
- Builds the processor with fullwasm tag
- Runs the integration tests with FULLWASM_TEST=1 environment variable

#### WASM Benchmarking Script

Created `test-wasm-benchmarks.sh` that:
- Builds the WASM models if needed
- Builds the processor with fullwasm tag
- Runs the benchmark tests comparing WASM vs stub
- Generates a detailed benchmark report with analysis

### 4. Documentation Updates

Updated `docs/examples/wasm-integration.md` to include:
- Information about testing the WASM implementation
- Details about performance benchmarking
- Performance considerations and tradeoffs
- Best practices for optimizing WASM models

## Test Coverage

The implemented tests cover:

1. **Functional Requirements**
   - WASM model loading
   - WASM function invocation
   - Error handling for invalid inputs
   - Input/output data handling
   - AI attribute annotation

2. **Non-Functional Requirements**
   - Performance (execution time)
   - Resource usage (memory)
   - Concurrency handling
   - Error resilience
   - Integration with OpenTelemetry data model

3. **Edge Cases**
   - Handling different types of errors
   - Processing very long traces
   - Handling missing attributes
   - Parallel processing of many spans

## Usage Instructions

To run the integration tests:

```bash
./test-fullwasm-integration.sh
```

To run performance benchmarks and compare WASM vs stub:

```bash
./test-wasm-benchmarks.sh
```

Benchmark results are saved to the `benchmark-results` directory for analysis.

## Next Steps

1. **CI/CD Integration**:
   - Add the integration tests to CI/CD pipelines
   - Ensure tests run on PRs to catch regressions

2. **Additional Test Cases**:
   - Test more complex WASM models
   - Add more edge case testing
   - Test reload functionality for WASM models

3. **Performance Tuning**:
   - Analyze benchmark results to identify bottlenecks
   - Optimize WASM models for better performance
   - Improve caching strategies

4. **Production Deployment Testing**:
   - Create deployment tests with actual telemetry data
   - Test under high load conditions
   - Set up long-running stability tests