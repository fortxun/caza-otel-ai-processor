# WASM Model JSON Schemas

This directory contains JSON Schema definitions for the input and output formats of each WASM model:

1. **error-classifier.json**: Schema for the Error Classifier model
2. **importance-sampler.json**: Schema for the Importance Sampler model
3. **entity-extractor.json**: Schema for the Entity Extractor model

## Usage

These schemas serve multiple purposes:

1. **Documentation**: They precisely define the expected input and output formats
2. **Validation**: The Go processor can use these schemas to validate data before/after model invocation
3. **Development**: IDE integration for validation during model development
4. **Testing**: Test cases can be validated against these schemas

## Example Validation

```javascript
// Node.js example of schema validation
const Ajv = require('ajv');
const ajv = new Ajv();

const schema = require('./error-classifier.json');
const validate = ajv.compile(schema);

const data = {
  input: {
    name: "ExecuteQuery",
    status: "Connection refused to database",
    kind: "CLIENT",
    attributes: { 
      "db.system": "postgresql", 
      "db.name": "users" 
    },
    resource: { 
      "service.name": "user-service" 
    }
  },
  output: {
    category: "database_error",
    system: "postgresql",
    owner: "database-team",
    severity: "high",
    impact: "medium",
    confidence: 0.92
  }
};

const valid = validate(data);
if (!valid) {
  console.log(validate.errors);
}
```

## Schema Details

Each schema defines:

- Input parameters and their types
- Required vs. optional fields
- Output structure and field types
- Enumerated values where applicable
- Value constraints (e.g., confidence scores between 0.0-1.0)

These schemas help ensure consistency between the Go processor and the WASM models.