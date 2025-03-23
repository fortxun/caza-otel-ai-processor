# WASM Integration Guide

This document explains how the CAZA OpenTelemetry AI Processor integrates with WebAssembly (WASM) models for AI-enhanced telemetry processing.

## Overview

The CAZA OpenTelemetry AI Processor uses WebAssembly (WASM) to provide AI-powered telemetry processing capabilities. It integrates with AssemblyScript-compiled WASM modules using the wasmer-go runtime.

## WASM Models

The processor comes with three pre-built WASM models:

1. **Error Classifier**: Analyzes error patterns and classifies errors by type, severity, and suggested owner.
2. **Importance Sampler**: Determines the importance of telemetry items for intelligent sampling.
3. **Entity Extractor**: Extracts meaningful entities from telemetry data for enhanced context.

## Building WASM Models

The WASM models are built from AssemblyScript source code. To build or update these models:

1. Ensure you have Node.js and npm installed.
2. Navigate to the `wasm-models` directory.
3. Install dependencies: `npm install`
4. Run the build script: `./build-models.sh` or `npm run asbuild`

This will compile the AssemblyScript source code into WASM modules and copy them to the `models` directory.

## WASM Runtime Implementation

The WASM runtime is implemented in `pkg/runtime/wasm_runtime.go` and provides:

1. **Model Loading**: Loads WASM models from files and instantiates them with wasmer-go.
2. **Function Invocation**: Provides methods to invoke functions in the WASM modules.
3. **Error Handling**: Handles errors from WASM function invocation.
4. **Result Caching**: Caches results for performance optimization.

### Important WASM Integration Features:

#### AssemblyScript Compatibility

The WASM runtime must provide certain environment functions required by AssemblyScript:

```go
// Create required abort function for AssemblyScript
abortFn := wasmer.NewFunction(
    store,
    wasmer.NewFunctionType(
        wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), 
        wasmer.NewValueTypes(),
    ),
    func(args []wasmer.Value) ([]wasmer.Value, error) {
        // Log the abort information
        msgPtr := args[0].I32()
        filePtr := args[1].I32()
        line := args[2].I32()
        col := args[3].I32()
        fmt.Printf("AssemblyScript abort called: msg=%d file=%d line=%d col=%d\n", 
            msgPtr, filePtr, line, col)
        return []wasmer.Value{}, nil
    },
)

// Register the required imports
importObject.Register(
    "env", 
    map[string]wasmer.IntoExtern{
        "abort": abortFn,
    },
)
```

#### Function Invocation

The WASM runtime invokes functions in the WASM modules:

```go
func (r *WasmRuntime) invokeWasmFunction(instance *wasmer.Instance, functionName, input string) (string, error) {
    // Get the function from the instance
    function, err := instance.Exports.GetFunction(functionName)
    if err != nil {
        return "", fmt.Errorf("function %s not found: %w", functionName, err)
    }

    // Invoke the function with the input
    result, err := function(input)
    if err != nil {
        return "", fmt.Errorf("failed to invoke function %s: %w", functionName, err)
    }

    // Convert the result to a string
    resultStr, ok := result.(string)
    if !ok {
        return "", fmt.Errorf("unexpected result type from function %s", functionName)
    }

    return resultStr, nil
}
```

## Configuration

The WASM runtime and models are configured in the processor configuration:

```yaml
processors:
  ai_processor:
    models:
      error_classifier:
        path: "./models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
      importance_sampler:
        path: "./models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30
      entity_extractor:
        path: "./models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50
    processing:
      batch_size: 50
      concurrency: 4
      queue_size: 1000
      timeout_ms: 500
      enable_parallel_processing: true
      max_parallel_workers: 8
      attribute_cache_size: 1000
      resource_cache_size: 100
      model_cache_results: true
      model_results_cache_size: 1000
```

### Configuration Options:

- **path**: Path to the WASM model file.
- **memory_limit_mb**: Memory limit for the WASM module in MB.
- **timeout_ms**: Timeout for WASM function execution in milliseconds.
- **model_cache_results**: Whether to cache model results.
- **model_results_cache_size**: Size of the model results cache.

## Testing

The processor includes comprehensive tests for both the stub and full WASM implementations:

### Integration Testing

To test the full WASM implementation:

```bash
./test-fullwasm-integration.sh
```

This script:
1. Builds the WASM models
2. Builds the processor with full WASM support
3. Runs integration tests that use actual WASM models

The integration tests verify:
- Proper loading of WASM models
- Correct function invocation
- Input/output data handling
- Error handling
- Parallel processing capabilities

### Performance Benchmarking

To compare the performance of the WASM implementation versus the stub implementation:

```bash
./test-wasm-benchmarks.sh
```

This script runs benchmarks that compare:
- Error classification performance
- Telemetry sampling performance
- Entity extraction performance
- Full pipeline performance

The benchmark results are saved to a report in the `benchmark-results` directory.

## Building with WASM Support

To build the processor with full WASM support:

```bash
./build-full-wasm.sh
```

This will:
1. Build the WASM models using AssemblyScript
2. Download the required dependencies
3. Build the processor with WASM support

## Troubleshooting

### Common Issues:

1. **"Missing import: `env`.`abort`"**: The WASM runtime needs to provide the `env.abort` function for AssemblyScript compatibility.

2. **"Failed to read WASM file"**: Check the paths in the configuration file. Ensure the WASM files exist at the specified path.

3. **"Failed to instantiate WASM module"**: Check the wasmer-go version and ensure it's compatible with the WASM modules.

4. **"Function not found"**: Ensure the WASM modules export the expected functions (`classify_error`, `sample_telemetry`, `extract_entities`).

## Advanced Topics

### Custom WASM Models

You can create custom WASM models to extend the functionality of the processor:

1. Create a new AssemblyScript project in the `wasm-models` directory.
2. Implement the required functions for your model.
3. Build the model with `npm run asbuild`.
4. Configure the processor to use your model.

### Memory Management

WASM modules have limited memory access. The AssemblyScript runtime handles memory allocation and deallocation. The `memory_limit_mb` configuration option controls the maximum memory available to the WASM module.

### Performance Optimization

To optimize performance:

1. **Enable Result Caching**: Set `model_cache_results: true` in the configuration.
2. **Tune Cache Size**: Adjust `model_results_cache_size` based on your use case.
3. **Parallel Processing**: Enable parallel processing with `enable_parallel_processing: true`.

### Performance Considerations

Based on benchmark results, the WASM implementation typically has:
- Higher startup time compared to the stub implementation
- Slightly higher per-operation latency
- Increased memory usage due to the WASM runtime

However, the WASM implementation offers significant advantages:
- Dynamic loading of models without recompilation
- Runtime model updates without processor restart
- Better separation of AI model code from processor code
- Enhanced security through WASM sandbox

For production deployment, consider these tradeoffs and choose the appropriate implementation based on your requirements.