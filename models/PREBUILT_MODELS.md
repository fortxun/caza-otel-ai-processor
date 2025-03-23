# Prebuilt WASM Models for AI-Enhanced Telemetry Processor

This directory should contain the prebuilt WebAssembly (WASM) models used by the AI-Enhanced Telemetry Processor.

## Required Model Files

The following WASM model files should be placed in this directory:

1. **error-classifier.wasm**: Error classification model
2. **importance-sampler.wasm**: Smart sampling model
3. **entity-extractor.wasm**: Entity extraction model

## Building the Models

The models can be built from source using the AssemblyScript compiler. To build the models:

1. Navigate to the `wasm-models` directory:
   ```
   cd ../wasm-models
   ```

2. Run the build script:
   ```
   ./build-models.sh
   ```

This will build the models and copy the WASM files to this directory.

## Model Versions

The current implementation uses these model versions:

- Error Classifier: v0.1.0
- Importance Sampler: v0.1.0
- Entity Extractor: v0.1.0

For production use, you should replace these models with trained, optimized versions.

## Model Validation

To verify that the models are working correctly:

1. Navigate to the `wasm-models` directory:
   ```
   cd ../wasm-models
   ```

2. Run the tests:
   ```
   npm run test
   ```

This will execute test cases for each model and verify that they produce the expected output.

## Configuring the Processor

In your OpenTelemetry Collector configuration YAML, reference these model files as follows:

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
      entity_extractor:
        path: "/models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50
```

## Security Considerations

These WASM models run in a sandboxed environment, but you should still:

1. Verify the integrity of the WASM files (checksums provided below)
2. Ensure the models are loaded from a trusted source
3. Apply appropriate memory limits as shown in the configuration example

## File Checksums

For security verification, compare the checksums of your WASM files with these expected values:

```
# These checksums will be generated during the build process
error-classifier.wasm: <MD5_CHECKSUM>
importance-sampler.wasm: <MD5_CHECKSUM>
entity-extractor.wasm: <MD5_CHECKSUM>
```

Note: Actual checksums will be generated when the models are built.