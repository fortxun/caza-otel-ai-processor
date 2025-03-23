# Model Customization Guide

This guide explains how to customize the WASM models used by the AI-Enhanced Telemetry Processor to better fit your specific needs.

## Understanding the WASM Models

The processor uses three types of WASM models:

1. **Error Classifier**: Categorizes errors and suggests owners
2. **Importance Sampler**: Makes intelligent sampling decisions
3. **Entity Extractor**: Identifies services, dependencies, and operations

Each model is implemented in AssemblyScript and compiled to WebAssembly.

## When to Customize Models

You might want to customize the models for several reasons:

- **Domain-specific patterns**: Add patterns specific to your technology stack
- **Organization-specific rules**: Customize owner assignments for your org chart
- **Performance optimization**: Optimize the models for your specific telemetry volume
- **Feature enhancement**: Add new capabilities to the models

## Customization Approach

### 1. Clone the Repository

```bash
git clone https://github.com/fortxun/caza-otel-ai-processor.git
cd caza-otel-ai-processor
```

### 2. Understand the Model Structure

Navigate to the model source code:

```bash
cd wasm-models
```

Each model has a similar structure:
- `assembly/index.ts`: Main model implementation
- `assembly/tsconfig.json`: TypeScript configuration
- `build/`: Output directory for compiled WASM

### 3. Modify the Model

Let's look at an example of customizing the Error Classifier model.

#### 3.1. Add Domain-Specific Error Patterns

Open `error-classifier/assembly/index.ts` and add your patterns:

```typescript
// Add your domain-specific patterns
const CUSTOM_ERROR_PATTERNS: string[] = [
  "lambda timeout", "stepfunction failed", "eks", "ecs",
  "health check failed", "disk full", "memory leak"
];

// Add custom system identifiers
if (text.includes("lambda") || text.includes("stepfunction")) {
  return "aws_serverless";
}

if (text.includes("eks") || text.includes("kubernetes")) {
  return "container_platform";
}
```

#### 3.2. Add Custom Owner Mappings

Customize owner assignments for your organization:

```typescript
// Update or add to the SERVICES map
SERVICES.set("aws_serverless", "cloud-team");
SERVICES.set("container_platform", "platform-team");
SERVICES.set("payment-gateway", "finance-team");
SERVICES.set("inventory", "warehouse-team");
```

### 4. Build the Customized Model

```bash
# Install dependencies
npm install

# Build a specific model
npm run asbuild:error-classifier

# Or build all models
npm run asbuild
```

### 5. Test the Customized Model

You can test the model using the provided test script:

```bash
node tests/index.js error-classifier
```

Or create a specific test case:

```javascript
// In tests/index.js
const testErrorClassifier = async () => {
  // Add a test for your custom pattern
  const customErrorInput = {
    name: "ProcessPayment",
    status: "Lambda timeout during payment processing",
    kind: "SERVER",
    attributes: { "aws.service": "lambda" },
    resource: { "service.name": "payment-service" }
  };

  const result = await runWasmFunction(
    "../error-classifier/build/error-classifier.wasm",
    "classify_error",
    JSON.stringify(customErrorInput)
  );

  console.log("Custom error classification:", result);
};
```

### 6. Deploy the Customized Model

Copy the compiled WASM file to your deployment:

```bash
cp error-classifier/build/error-classifier.wasm /path/to/your/models/directory/
```

## Advanced Customization Techniques

### Adding New Functions

You can add new functions to the models:

```typescript
// Add a new export function
export function analyze_trend(inputJson: string): string {
  // Implementation...
  return JSON.stringify({ trend: "increasing", confidence: 0.85 });
}
```

Remember to update the model interface documentation and the processor code to use the new function.

### Using Additional Libraries

You can use additional AssemblyScript libraries:

```bash
npm install --save as-string-sink
```

Then import them in your model:

```typescript
import { StringSink } from "as-string-sink";
```

### Performance Optimization

Some techniques for optimizing model performance:

1. **Simplify pattern matching**: Use fewer, more effective patterns
2. **Optimize string operations**: String operations are expensive in WASM
3. **Pre-compute lookup tables**: For frequently accessed data
4. **Minimize memory allocations**: Reuse objects where possible

## Example: Custom Importance Sampler

Here's a more complete example of customizing the Importance Sampler:

```typescript
// in importance-sampler/assembly/index.ts

// Add custom sampling rules
const HIGH_VALUE_OPERATIONS: string[] = [
  "checkout", "payment", "user_registration", "login"
];

const HIGH_VALUE_SERVICES: string[] = [
  "cart-service", "payment-gateway", "user-service", "auth-service"
];

/**
 * Determine if a span relates to a high-value operation
 */
function isHighValueOperation(name: string, attributes: Map<string, string>): boolean {
  // Check operation name
  for (let i = 0; i < HIGH_VALUE_OPERATIONS.length; i++) {
    if (name.toLowerCase().includes(HIGH_VALUE_OPERATIONS[i])) {
      return true;
    }
  }
  
  // Check service name
  const serviceName = attributes.get("service.name") || "";
  for (let i = 0; i < HIGH_VALUE_SERVICES.length; i++) {
    if (serviceName.toLowerCase().includes(HIGH_VALUE_SERVICES[i])) {
      return true;
    }
  }
  
  return false;
}

/**
 * Modify the sample_telemetry function
 */
export function sample_telemetry(inputJson: string): string {
  // ... existing code ...
  
  // Add custom sampling logic
  if (isHighValueOperation(spanName, attributes)) {
    importance = 0.9;
    keep = true;
    reason = "high_value_operation";
  }
  
  // ... rest of the function ...
}
```

## Example: Custom Entity Extractor

Customize the Entity Extractor to recognize your specific services:

```typescript
// in entity-extractor/assembly/index.ts

// Add custom service patterns
const CUSTOM_SERVICES: Map<string, string[]> = new Map<string, string[]>();
CUSTOM_SERVICES.set("payment", ["stripe", "paypal", "braintree", "checkout"]);
CUSTOM_SERVICES.set("storage", ["s3", "blob", "file", "document"]);
CUSTOM_SERVICES.set("search", ["elasticsearch", "algolia", "solr", "search-api"]);

/**
 * Enhance service detection with custom patterns
 */
function detectCustomServices(text: string): string[] {
  const result: string[] = [];
  
  const keys = CUSTOM_SERVICES.keys();
  for (let i = 0; i < keys.length; i++) {
    const category = keys[i];
    const patterns = CUSTOM_SERVICES.get(category) || [];
    
    for (let j = 0; j < patterns.length; j++) {
      if (text.toLowerCase().includes(patterns[j])) {
        result.push(category + "-service");
        break;
      }
    }
  }
  
  return result;
}

/**
 * Modify the extract_entities function
 */
export function extract_entities(inputJson: string): string {
  // ... existing code ...
  
  // Add custom entity detection
  const customServices = detectCustomServices(combinedText);
  for (let i = 0; i < customServices.length; i++) {
    services.push(customServices[i]);
  }
  
  // ... rest of the function ...
}
```

## Testing and Validation

Always test your customized models thoroughly:

1. **Unit testing**: Test each function in isolation
2. **Integration testing**: Test the model with the processor
3. **Performance testing**: Benchmark the model to ensure it meets performance targets
4. **Validation**: Verify that the model output conforms to the expected schema

## Best Practices

1. **Documentation**: Document your customizations for future maintainers
2. **Version Control**: Keep your customized models in version control
3. **Modularity**: Structure your code so custom parts are separate from core functionality
4. **Configuration**: Make your customizations configurable where possible
5. **Testing**: Create specific tests for your custom logic
6. **Backward Compatibility**: Ensure your changes maintain the expected interfaces