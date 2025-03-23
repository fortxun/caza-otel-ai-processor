const fs = require('fs');
const path = require('path');
const loader = require('@assemblyscript/loader');

// Test cases for each model
const errorClassifierTests = [
  {
    name: "Database error test",
    input: {
      name: "ExecuteQuery",
      status: "Connection refused to database",
      kind: "CLIENT",
      attributes: { 
        "db.system": "postgresql", 
        "db.name": "users",
        "db.statement": "SELECT * FROM users" 
      },
      resource: { 
        "service.name": "user-service" 
      }
    }
  },
  {
    name: "Authentication error test",
    input: {
      name: "Authenticate",
      status: "Invalid token provided",
      kind: "SERVER",
      attributes: { 
        "http.status_code": "401",
        "http.method": "POST",
        "http.url": "/api/login" 
      },
      resource: { 
        "service.name": "auth-service" 
      }
    }
  }
];

const importanceSamplerTests = [
  {
    name: "High importance span test",
    input: {
      name: "processPayment",
      status: "OK",
      kind: "CLIENT",
      duration: 150,
      attributes: {
        "http.method": "POST",
        "http.url": "/api/payments",
        "http.status_code": "200"
      },
      resource: {
        "service.name": "payment-service"
      }
    }
  },
  {
    name: "Low importance span test",
    input: {
      name: "getHealthCheck",
      status: "OK",
      kind: "CLIENT",
      duration: 5,
      attributes: {
        "http.method": "GET",
        "http.url": "/health",
        "http.status_code": "200"
      },
      resource: {
        "service.name": "monitoring-service"
      }
    }
  },
  {
    name: "Error span test",
    input: {
      name: "getUser",
      status: "Error",
      kind: "CLIENT",
      duration: 50,
      attributes: {
        "http.method": "GET",
        "http.url": "/api/users/123",
        "http.status_code": "500",
        "error.message": "Internal server error"
      },
      resource: {
        "service.name": "user-service"
      }
    }
  }
];

const entityExtractorTests = [
  {
    name: "API service test",
    input: {
      name: "handleUserRequest",
      description: "Process user API request",
      type: "span",
      attributes: {
        "http.method": "POST",
        "http.url": "/api/users",
        "http.status_code": "201"
      },
      resource: {
        "service.name": "user-api-service"
      }
    }
  },
  {
    name: "Database operation test",
    input: {
      name: "queryUserDatabase",
      description: "Query user information from database",
      type: "span",
      attributes: {
        "db.system": "mysql",
        "db.name": "users",
        "db.statement": "SELECT * FROM users WHERE id = ?"
      },
      resource: {
        "service.name": "user-service"
      }
    }
  }
];

async function runTests() {
  console.log("WASM Model Tests\n");
  
  // Test Error Classifier
  console.log("Testing Error Classifier Model...");
  const errorClassifierPath = path.resolve(__dirname, '../error-classifier/build/error-classifier.wasm');
  if (fs.existsSync(errorClassifierPath)) {
    const errorClassifierModule = await loader.instantiate(fs.readFileSync(errorClassifierPath));
    
    for (const test of errorClassifierTests) {
      console.log(`\n  ${test.name}:`);
      console.log(`  Input: ${JSON.stringify(test.input, null, 2)}`);
      
      try {
        const result = errorClassifierModule.exports.classify_error(JSON.stringify(test.input));
        console.log(`  Result: ${result}`);
      } catch (err) {
        console.error(`  Error: ${err.message}`);
      }
    }
  } else {
    console.log("  Error classifier WASM not found. Build it first with 'npm run asbuild:error-classifier'");
  }
  
  // Test Importance Sampler
  console.log("\nTesting Importance Sampler Model...");
  const importanceSamplerPath = path.resolve(__dirname, '../importance-sampler/build/importance-sampler.wasm');
  if (fs.existsSync(importanceSamplerPath)) {
    const importanceSamplerModule = await loader.instantiate(fs.readFileSync(importanceSamplerPath));
    
    for (const test of importanceSamplerTests) {
      console.log(`\n  ${test.name}:`);
      console.log(`  Input: ${JSON.stringify(test.input, null, 2)}`);
      
      try {
        const result = importanceSamplerModule.exports.sample_telemetry(JSON.stringify(test.input));
        console.log(`  Result: ${result}`);
      } catch (err) {
        console.error(`  Error: ${err.message}`);
      }
    }
  } else {
    console.log("  Importance sampler WASM not found. Build it first with 'npm run asbuild:importance-sampler'");
  }
  
  // Test Entity Extractor
  console.log("\nTesting Entity Extractor Model...");
  const entityExtractorPath = path.resolve(__dirname, '../entity-extractor/build/entity-extractor.wasm');
  if (fs.existsSync(entityExtractorPath)) {
    const entityExtractorModule = await loader.instantiate(fs.readFileSync(entityExtractorPath));
    
    for (const test of entityExtractorTests) {
      console.log(`\n  ${test.name}:`);
      console.log(`  Input: ${JSON.stringify(test.input, null, 2)}`);
      
      try {
        const result = entityExtractorModule.exports.extract_entities(JSON.stringify(test.input));
        console.log(`  Result: ${result}`);
      } catch (err) {
        console.error(`  Error: ${err.message}`);
      }
    }
  } else {
    console.log("  Entity extractor WASM not found. Build it first with 'npm run asbuild:entity-extractor'");
  }
}

// Run the tests
runTests().catch(console.error);