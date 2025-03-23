# WASM Models for AI-Enhanced Telemetry Processor

This directory contains WebAssembly (WASM) models used by the AI-Enhanced Telemetry Processor.

## Required Models

### 1. Error Classifier (`error-classifier.wasm`)

- **Purpose**: Categorize errors, identify affected systems, and suggest owners
- **Input**: Error messages, stack traces, and context
- **Output**: Error category, affected system, suggested owner
- **Size Target**: Under 10MB in WASM format

### 2. Telemetry Sampler (`importance-sampler.wasm`)

- **Purpose**: Make smart sampling decisions based on telemetry content
- **Input**: Spans, logs, or metrics with attributes
- **Output**: Importance score, sampling decision
- **Size Target**: Under 8MB in WASM format

### 3. Entity Extractor (`entity-extractor.wasm`)

- **Purpose**: Identify services, dependencies, and operations from telemetry
- **Input**: Structured or unstructured telemetry
- **Output**: Identified services, dependencies, operations
- **Size Target**: Under 15MB in WASM format

## Model Development

These models are typically developed using the following approach:

1. Start with specialized, small models rather than one large model
2. Use distillation techniques from larger LLMs
3. Quantize models to INT8 precision
4. Apply domain-specific fine-tuning on telemetry data
5. Package models in WASM format with standard input/output interfaces

## Placeholder Models

The current models in this directory are placeholders for testing purposes. For production use, you should replace them with trained models that implement the required functionality.

## Interface Requirements

Each model should implement specific functions for its intended purpose:

### Error Classifier Interface

```javascript
// classify_error takes a JSON string with error information and returns a JSON string with error classification
function classify_error(jsonInput) {
  // Process input and return classification
  return JSON.stringify({
    category: "database_connection",
    system: "postgres",
    owner: "database-team",
    severity: "high",
    confidence: 0.92
  });
}
```

### Telemetry Sampler Interface

```javascript
// sample_telemetry takes a JSON string with telemetry information and returns a JSON string with sampling decision
function sample_telemetry(jsonInput) {
  // Process input and return sampling decision
  return JSON.stringify({
    importance: 0.75,
    keep: true,
    reason: "contains_key_component"
  });
}
```

### Entity Extractor Interface

```javascript
// extract_entities takes a JSON string with telemetry information and returns a JSON string with identified entities
function extract_entities(jsonInput) {
  // Process input and return entities
  return JSON.stringify({
    services: ["order-service", "payment-gateway"],
    dependencies: ["redis", "kafka"],
    operations: ["checkout", "payment-processing"],
    confidence: 0.88
  });
}
```