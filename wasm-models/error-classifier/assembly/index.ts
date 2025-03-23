/**
 * Error Classifier WASM Module
 * 
 * This module classifies error telemetry data into meaningful categories,
 * extracts entities, suggests potential owners, and estimates severity.
 */

// Import AssemblyScript runtime
import { JSON } from "assemblyscript-json/assembly";

// Database error patterns
const DB_ERROR_PATTERNS: string[] = [
  "connection refused", "timeout", "deadlock", "constraint violation",
  "duplicate key", "foreign key", "sql syntax", "database", "postgres",
  "mysql", "mongodb", "redis", "connection pool", "query failed"
];

// Network error patterns
const NETWORK_ERROR_PATTERNS: string[] = [
  "connection refused", "timeout", "host unreachable", "dns lookup",
  "socket error", "network", "http error", "tcp", "ssl", "tls",
  "connection reset", "EOF", "i/o timeout"
];

// Authentication error patterns
const AUTH_ERROR_PATTERNS: string[] = [
  "unauthorized", "forbidden", "permission denied", "access denied",
  "not authenticated", "invalid token", "expired token", "auth", "password",
  "credentials", "oauth", "session expired", "login failed"
];

// Configuration error patterns
const CONFIG_ERROR_PATTERNS: string[] = [
  "configuration", "config", "missing parameter", "invalid setting",
  "environment variable", "env var", "flag", "option", "property"
];

// Rate limiting error patterns
const RATE_LIMIT_PATTERNS: string[] = [
  "rate limit", "throttle", "too many requests", "quota exceeded",
  "429", "limit reached"
];

// Service names
const SERVICES: Map<string, string> = new Map<string, string>();
SERVICES.set("postgres", "database-team");
SERVICES.set("mysql", "database-team");
SERVICES.set("mongodb", "database-team");
SERVICES.set("redis", "cache-team");
SERVICES.set("auth", "security-team");
SERVICES.set("login", "security-team");
SERVICES.set("payment", "billing-team");
SERVICES.set("api", "platform-team");
SERVICES.set("frontend", "ui-team");
SERVICES.set("backend", "backend-team");
SERVICES.set("order", "orders-team");
SERVICES.set("user", "user-team");
SERVICES.set("account", "user-team");
SERVICES.set("config", "platform-team");
SERVICES.set("gateway", "platform-team");

/**
 * Classify an error based on its message and context
 * Input is a JSON string with error details
 * Output is a JSON string with classification
 */
export function classify_error(inputJson: string): string {
  // Parse the input JSON
  const jsonObj = <JSON.Obj>JSON.parse(inputJson);
  
  // Extract error information
  const errorMessage = getStringValue(jsonObj, "status") || "";
  const errorName = getStringValue(jsonObj, "name") || "";
  const errorAttributes = getMapValue(jsonObj, "attributes");
  const resourceAttributes = getMapValue(jsonObj, "resource");
  
  // Combine all text for pattern matching
  const combinedText = (errorMessage + " " + errorName).toLowerCase();
  
  // Classify error category
  const category = determineCategory(combinedText);
  
  // Determine affected system
  const system = determineSystem(combinedText, errorAttributes, resourceAttributes);
  
  // Suggest owner
  const owner = suggestOwner(system, category, errorAttributes, resourceAttributes);
  
  // Estimate severity
  const severity = estimateSeverity(combinedText, category);
  
  // Estimate business impact
  const impact = estimateImpact(system, severity);
  
  // Create and return the classification result
  const result: JSON.Obj = new JSON.Obj();
  result.set("category", category);
  result.set("system", system);
  result.set("owner", owner);
  result.set("severity", severity);
  result.set("impact", impact);
  result.set("confidence", 0.85);
  
  return result.toString();
}

/**
 * Determine the error category based on the error text
 */
function determineCategory(text: string): string {
  // Check for database errors
  for (let i = 0; i < DB_ERROR_PATTERNS.length; i++) {
    if (text.includes(DB_ERROR_PATTERNS[i])) {
      return "database_error";
    }
  }
  
  // Check for network errors
  for (let i = 0; i < NETWORK_ERROR_PATTERNS.length; i++) {
    if (text.includes(NETWORK_ERROR_PATTERNS[i])) {
      return "network_error";
    }
  }
  
  // Check for authentication errors
  for (let i = 0; i < AUTH_ERROR_PATTERNS.length; i++) {
    if (text.includes(AUTH_ERROR_PATTERNS[i])) {
      return "authentication_error";
    }
  }
  
  // Check for configuration errors
  for (let i = 0; i < CONFIG_ERROR_PATTERNS.length; i++) {
    if (text.includes(CONFIG_ERROR_PATTERNS[i])) {
      return "configuration_error";
    }
  }
  
  // Check for rate limiting errors
  for (let i = 0; i < RATE_LIMIT_PATTERNS.length; i++) {
    if (text.includes(RATE_LIMIT_PATTERNS[i])) {
      return "rate_limiting_error";
    }
  }
  
  // Default category
  return "unclassified_error";
}

/**
 * Determine the affected system from the error
 */
function determineSystem(text: string, attributes: Map<string, string>, resourceAttributes: Map<string, string>): string {
  // Check service name from resource attributes
  const serviceName = resourceAttributes.get("service.name");
  if (serviceName) {
    return serviceName;
  }
  
  // Check for common systems in the error text
  if (text.includes("postgres") || text.includes("postgresql")) {
    return "postgres";
  }
  
  if (text.includes("mysql")) {
    return "mysql";
  }
  
  if (text.includes("mongodb") || text.includes("mongo")) {
    return "mongodb";
  }
  
  if (text.includes("redis")) {
    return "redis";
  }
  
  if (text.includes("kafka")) {
    return "kafka";
  }
  
  if (text.includes("http") || text.includes("api") || text.includes("rest")) {
    return "api_service";
  }
  
  // Default system
  return "unknown_system";
}

/**
 * Suggest an owner for the error
 */
function suggestOwner(system: string, category: string, attributes: Map<string, string>, resourceAttributes: Map<string, string>): string {
  // Check for owner in service name
  for (let i = 0; i < SERVICES.keys().length; i++) {
    const key = SERVICES.keys()[i];
    if (system.includes(key)) {
      return SERVICES.get(key) || "unknown-team";
    }
  }
  
  // Assign owner based on category
  if (category === "database_error") {
    return "database-team";
  }
  
  if (category === "network_error") {
    return "infrastructure-team";
  }
  
  if (category === "authentication_error") {
    return "security-team";
  }
  
  if (category === "configuration_error") {
    return "platform-team";
  }
  
  // Default owner
  return "unknown-team";
}

/**
 * Estimate error severity
 */
function estimateSeverity(text: string, category: string): string {
  // Critical errors
  if (text.includes("critical") || text.includes("fatal") || text.includes("crash")) {
    return "critical";
  }
  
  // High severity errors
  if (text.includes("exception") || text.includes("failure") || 
      category === "database_error" || category === "authentication_error") {
    return "high";
  }
  
  // Medium severity errors
  if (text.includes("error") || text.includes("warning") || 
      category === "network_error" || category === "configuration_error") {
    return "medium";
  }
  
  // Default severity
  return "low";
}

/**
 * Estimate business impact
 */
function estimateImpact(system: string, severity: string): string {
  // Critical systems with high severity
  if ((system === "postgres" || system === "mysql" || system === "api_service") && 
      (severity === "critical" || severity === "high")) {
    return "high";
  }
  
  // Critical systems with medium severity
  if ((system === "postgres" || system === "mysql" || system === "api_service") && 
      severity === "medium") {
    return "medium";
  }
  
  // Non-critical systems with high severity
  if ((system === "redis" || system === "kafka") && 
      (severity === "critical" || severity === "high")) {
    return "medium";
  }
  
  // Default impact
  return "low";
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