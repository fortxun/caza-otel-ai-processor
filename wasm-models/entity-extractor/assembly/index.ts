/**
 * Entity Extractor WASM Module
 * 
 * This module extracts entities like services, dependencies, and operations
 * from telemetry data to enhance context and enable better analysis.
 */

// Import AssemblyScript runtime
import { JSON } from "assemblyscript-json/assembly";

// Service patterns for identification
const SERVICE_PATTERNS: string[] = [
  "service", "api", "gateway", "backend", "frontend", "server",
  "app", "application", "system", "platform", "module"
];

// Database dependency patterns
const DB_DEPENDENCY_PATTERNS: string[] = [
  "postgres", "mysql", "mongodb", "db", "database", "redis", "memcached",
  "cassandra", "dynamodb", "sql", "nosql", "jdbc", "connection pool"
];

// Queue/messaging dependency patterns
const QUEUE_DEPENDENCY_PATTERNS: string[] = [
  "kafka", "rabbitmq", "sqs", "queue", "pubsub", "nats", "eventbus",
  "event hub", "mq", "jms", "messaging", "stream", "kinesis"
];

// Storage dependency patterns
const STORAGE_DEPENDENCY_PATTERNS: string[] = [
  "s3", "blob", "storage", "file", "disk", "volume", "bucket", "gcs", "adls"
];

// External API dependency patterns
const API_DEPENDENCY_PATTERNS: string[] = [
  "http", "https", "rest", "api", "endpoint", "url", "uri", "client", 
  "request", "graphql", "grpc", "external"
];

// Common operation types
const OPERATION_TYPES: string[] = [
  "create", "read", "update", "delete", "get", "post", "put", "patch", "query",
  "search", "list", "insert", "modify", "remove", "process", "compute", "login",
  "authenticate", "authorize", "check", "validate", "sync", "handle"
];

// Common operation objects
const OPERATION_OBJECTS: string[] = [
  "user", "account", "order", "product", "payment", "subscription", "customer",
  "transaction", "invoice", "request", "message", "event", "document", "record",
  "item", "file", "data", "resource", "session", "token", "credentials"
];

/**
 * Extract entities from telemetry data
 * Input is a JSON string with telemetry details
 * Output is a JSON string with extracted entities
 */
export function extract_entities(inputJson: string): string {
  // Parse the input JSON
  const jsonObj = <JSON.Obj>JSON.parse(inputJson);
  
  // Extract telemetry information
  const name = getStringValue(jsonObj, "name") || "";
  const description = getStringValue(jsonObj, "description") || "";
  const type = getStringValue(jsonObj, "type") || "";
  const body = getStringValue(jsonObj, "body") || "";
  const attributes = getMapValue(jsonObj, "attributes");
  const resourceAttributes = getMapValue(jsonObj, "resource");
  
  // Combine all text for analysis
  const combinedText = (
    name + " " + description + " " + type + " " + body
  ).toLowerCase();
  
  // Extract services
  const services = extractServices(combinedText, resourceAttributes);
  
  // Extract dependencies
  const dependencies = extractDependencies(combinedText, attributes);
  
  // Extract operations
  const operations = extractOperations(combinedText, name);
  
  // Calculate confidence
  const confidence = calculateConfidence(services, dependencies, operations);
  
  // Create result object with extracted entities
  const result: JSON.Obj = new JSON.Obj();
  result.set("services", arrayToJsonString(services));
  result.set("dependencies", arrayToJsonString(dependencies));
  result.set("operations", arrayToJsonString(operations));
  result.set("confidence", confidence);
  
  return result.toString();
}

/**
 * Extract services from telemetry data
 */
function extractServices(text: string, resourceAttributes: Map<string, string>): string[] {
  const services: string[] = [];
  
  // First check resource attributes for service name
  const serviceName = resourceAttributes.get("service.name");
  if (serviceName) {
    services.push(serviceName);
  }
  
  // Look for service patterns in the text
  for (let i = 0; i < SERVICE_PATTERNS.length; i++) {
    const pattern = SERVICE_PATTERNS[i];
    
    // Check for pattern-name format (e.g., "service-name", "api-users")
    const regex = pattern + "[- _][a-zA-Z0-9]+";
    const matches = findMatches(text, regex);
    
    for (let j = 0; j < matches.length; j++) {
      if (!services.includes(matches[j])) {
        services.push(matches[j]);
      }
    }
  }
  
  return services;
}

/**
 * Extract dependencies from telemetry data
 */
function extractDependencies(text: string, attributes: Map<string, string>): string[] {
  const dependencies: string[] = [];
  
  // Check attributes for dependencies
  const dbName = attributes.get("db.name");
  if (dbName) {
    dependencies.push(dbName);
  }
  
  const dbSystem = attributes.get("db.system");
  if (dbSystem) {
    dependencies.push(dbSystem);
  }
  
  const messagingSystem = attributes.get("messaging.system");
  if (messagingSystem) {
    dependencies.push(messagingSystem);
  }
  
  // Check for database dependencies
  for (let i = 0; i < DB_DEPENDENCY_PATTERNS.length; i++) {
    if (text.includes(DB_DEPENDENCY_PATTERNS[i])) {
      if (!dependencies.includes(DB_DEPENDENCY_PATTERNS[i])) {
        dependencies.push(DB_DEPENDENCY_PATTERNS[i]);
      }
    }
  }
  
  // Check for queue/messaging dependencies
  for (let i = 0; i < QUEUE_DEPENDENCY_PATTERNS.length; i++) {
    if (text.includes(QUEUE_DEPENDENCY_PATTERNS[i])) {
      if (!dependencies.includes(QUEUE_DEPENDENCY_PATTERNS[i])) {
        dependencies.push(QUEUE_DEPENDENCY_PATTERNS[i]);
      }
    }
  }
  
  // Check for storage dependencies
  for (let i = 0; i < STORAGE_DEPENDENCY_PATTERNS.length; i++) {
    if (text.includes(STORAGE_DEPENDENCY_PATTERNS[i])) {
      if (!dependencies.includes(STORAGE_DEPENDENCY_PATTERNS[i])) {
        dependencies.push(STORAGE_DEPENDENCY_PATTERNS[i]);
      }
    }
  }
  
  // Check for API dependencies
  for (let i = 0; i < API_DEPENDENCY_PATTERNS.length; i++) {
    if (text.includes(API_DEPENDENCY_PATTERNS[i])) {
      if (!dependencies.includes(API_DEPENDENCY_PATTERNS[i])) {
        dependencies.push(API_DEPENDENCY_PATTERNS[i]);
      }
    }
  }
  
  return dependencies;
}

/**
 * Extract operations from telemetry data
 */
function extractOperations(text: string, name: string): string[] {
  const operations: string[] = [];
  const lowerText = text.toLowerCase();
  const lowerName = name.toLowerCase();
  
  // Extract from span/metric name first
  // Look for verb-noun patterns (e.g., "createUser", "get_order", "updatePayment")
  for (let i = 0; i < OPERATION_TYPES.length; i++) {
    const verb = OPERATION_TYPES[i];
    
    for (let j = 0; j < OPERATION_OBJECTS.length; j++) {
      const noun = OPERATION_OBJECTS[j];
      
      // Check for verb+noun pattern with different separators
      const patterns = [
        verb + noun, 
        verb + "_" + noun, 
        verb + "-" + noun, 
        verb + " " + noun
      ];
      
      for (let k = 0; k < patterns.length; k++) {
        if (lowerName.includes(patterns[k]) || lowerText.includes(patterns[k])) {
          if (!operations.includes(verb + "_" + noun)) {
            operations.push(verb + "_" + noun);
          }
        }
      }
    }
  }
  
  // If no operations found from patterns, use the name itself
  if (operations.length === 0 && name.length > 0) {
    operations.push(name);
  }
  
  return operations;
}

/**
 * Calculate confidence score for entity extraction
 */
function calculateConfidence(services: string[], dependencies: string[], operations: string[]): f64 {
  let confidence: f64 = 0.5; // Start with base confidence
  
  // More entities found = higher confidence
  if (services.length > 0) {
    confidence += 0.1;
  }
  
  if (dependencies.length > 0) {
    confidence += 0.1;
  }
  
  if (operations.length > 0) {
    confidence += 0.1;
  }
  
  // More detailed extraction = higher confidence
  if (services.length > 1) {
    confidence += 0.05;
  }
  
  if (dependencies.length > 1) {
    confidence += 0.05;
  }
  
  if (operations.length > 1) {
    confidence += 0.05;
  }
  
  // Cap confidence at 0.95 (never be 100% confident)
  if (confidence > 0.95) {
    confidence = 0.95;
  }
  
  return confidence;
}

/**
 * Helper function to extract a string value from a JSON object
 */
function getStringValue(obj: JSON.Obj, key: string): string | null {
  const value = obj.get(key);
  if (value && value.isString) {
    const strVal = value as JSON.Str;
    return strVal.valueOf();
  }
  return null;
}

/**
 * Helper function to extract a map of attributes from a JSON object
 */
function getMapValue(obj: JSON.Obj, key: string): Map<string, string> {
  const result = new Map<string, string>();
  const value = obj.get(key);
  
  if (value && value.isObj) {
    const mapObj = <JSON.Obj>value;
    const keys = mapObj.keys;
    
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i];
      const v = mapObj.get(k);
      
      if (v) {
        if (v.isString) {
          const strVal = v as JSON.Str;
          result.set(k, strVal.valueOf());
        } else if (v.isInteger) {
          const intVal = v as JSON.Integer;
          result.set(k, intVal.valueOf().toString());
        } else if (v.isFloat) {
          const floatVal = v as JSON.Float;
          result.set(k, floatVal.valueOf().toString());
        } else if (v.isBool) {
          const boolVal = v as JSON.Bool;
          result.set(k, boolVal.valueOf().toString());
        }
      }
    }
  }
  
  return result;
}

/**
 * Helper function to convert a string array to a JSON array string
 */
function arrayToJsonString(arr: string[]): string {
  let result = "[";
  for (let i = 0; i < arr.length; i++) {
    result += "\"" + arr[i] + "\"";
    if (i < arr.length - 1) {
      result += ",";
    }
  }
  result += "]";
  return result;
}

/**
 * Find all matches of a regex pattern in a text
 * This is a simplified implementation since AssemblyScript has limited regex support
 */
function findMatches(text: string, pattern: string): string[] {
  const results: string[] = [];
  
  // This is a very simplified "regex" implementation
  // In a real implementation, use a proper regex library or implement more robust matching
  
  // For this demo, we'll just look for pattern-word combinations
  const words = text.split(" ");
  for (let i = 0; i < words.length; i++) {
    // Check if word contains our pattern
    if (words[i].includes(pattern)) {
      results.push(words[i]);
    }
  }
  
  return results;
}