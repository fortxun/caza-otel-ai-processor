# WASM Models for AI-Enhanced Telemetry Processor

This directory contains the source code and build configuration for the WebAssembly (WASM) models used by the OpenTelemetry AI Processor.

## Overview

The WASM models are written in AssemblyScript, a TypeScript-like language that compiles to WebAssembly. Each model is designed to perform a specific task:

1. **Error Classifier**: Categorizes errors and provides context about them
2. **Importance Sampler**: Determines which telemetry data to keep based on content
3. **Entity Extractor**: Identifies services, dependencies, and operations from telemetry

## Directory Structure

```
wasm-models/
├── error-classifier/        # Error classification model
│   ├── assembly/            # AssemblyScript source code
│   └── build/               # Compiled WASM files
├── importance-sampler/      # Importance sampling model
│   ├── assembly/            # AssemblyScript source code
│   └── build/               # Compiled WASM files
├── entity-extractor/        # Entity extraction model
│   ├── assembly/            # AssemblyScript source code
│   └── build/               # Compiled WASM files
├── package.json             # NPM package configuration
└── asconfig.json            # AssemblyScript compiler configuration
```

## Building the Models

To build all models:

```bash
cd wasm-models
npm install
npm run asbuild
```

To build a specific model:

```bash
npm run asbuild:error-classifier
npm run asbuild:importance-sampler
npm run asbuild:entity-extractor
```

## Model Interfaces

### Error Classifier

**Input**: JSON string containing:
- `name`: Error name
- `status`: Error message/status
- `kind`: Type of span/operation
- `attributes`: Map of attributes
- `resource`: Map of resource attributes

**Output**: JSON string containing:
- `category`: Error category (e.g., "database_error", "network_error")
- `system`: Affected system (e.g., "postgres", "api_service")
- `owner`: Suggested owner/team (e.g., "database-team")
- `severity`: Error severity (e.g., "high", "medium", "low")
- `impact`: Business impact (e.g., "high", "medium", "low")
- `confidence`: Confidence score (0.0-1.0)

### Importance Sampler

**Input**: JSON string containing:
- `name`: Span/metric/log name
- `status`: Status code/message
- `kind`: Type of span/operation
- `duration`: Duration in milliseconds (for spans)
- `attributes`: Map of attributes
- `resource`: Map of resource attributes

**Output**: JSON string containing:
- `importance`: Importance score (0.0-1.0)
- `keep`: Boolean decision to keep or drop
- `reason`: Reason for the decision (e.g., "error_status", "slow_duration")

### Entity Extractor

**Input**: JSON string containing:
- `name`: Span/metric/log name
- `description`: Description (if available)
- `type`: Type of telemetry
- `body`: Log body (for logs)
- `attributes`: Map of attributes
- `resource`: Map of resource attributes

**Output**: JSON string containing:
- `services`: Array of identified services
- `dependencies`: Array of identified dependencies
- `operations`: Array of identified operations
- `confidence`: Confidence score (0.0-1.0)

## Implementation Notes

### Pattern Matching

The current implementation uses simple pattern matching techniques to identify entities and classify errors. In a production environment, you would likely use more sophisticated techniques:

- Pre-trained machine learning models
- More robust regex pattern matching
- Embeddings-based similarity matching
- Transformer models fine-tuned on telemetry data

### Size Optimization

To meet the size targets specified in the project requirements:

- Models are kept small and specialized
- Minimal dependencies are used
- Only essential patterns are included
- Code is optimized for size

### Memory Usage

The models are designed to use minimal memory:

- No large lookup tables
- Efficient string processing
- Small working memory footprint
- Simple data structures

## Deployment

After building, copy the WASM files to the `/models` directory of the OpenTelemetry Collector:

```bash
cp error-classifier/build/error-classifier.wasm ../models/
cp importance-sampler/build/importance-sampler.wasm ../models/
cp entity-extractor/build/entity-extractor.wasm ../models/
```