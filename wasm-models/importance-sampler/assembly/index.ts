/**
 * Importance Sampler WASM Module
 * 
 * This module evaluates telemetry data and assigns an importance score
 * to make smart sampling decisions.
 */

// Import AssemblyScript runtime
import { JSON } from "assemblyscript-json/assembly";

// Important span name patterns
const IMPORTANT_SPAN_PATTERNS: string[] = [
  "checkout", "payment", "order", "create", "delete", "auth",
  "login", "register", "user", "authenticate", "purchase", "transaction"
];

// Important attribute keys
const IMPORTANT_ATTRIBUTE_KEYS: string[] = [
  "http.status_code", "error", "exception", "db.statement",
  "user.id", "customer.id", "order.id", "payment.id"
];

// Critical service names
const CRITICAL_SERVICES: string[] = [
  "payment-service", "checkout-service", "auth-service", "order-service",
  "user-service", "api-gateway", "inventory-service"
];

/**
 * Sample telemetry based on its content and context
 * Input is a JSON string with telemetry details
 * Output is a JSON string with sampling decision
 */
export function sample_telemetry(inputJson: string): string {
  // Parse the input JSON
  const jsonObj = <JSON.Obj>JSON.parse(inputJson);
  
  // Extract telemetry information
  const name = getStringValue(jsonObj, "name") || "";
  const status = getStringValue(jsonObj, "status") || "";
  const kind = getStringValue(jsonObj, "kind") || "";
  const durationMs = getNumberValue(jsonObj, "duration");
  const attributes = getMapValue(jsonObj, "attributes");
  const resourceAttributes = getMapValue(jsonObj, "resource");
  
  // Calculate importance score
  const importanceScore = calculateImportance(
    name, status, kind, durationMs, attributes, resourceAttributes
  );
  
  // Determine if we should keep this telemetry
  const keepDecision = determineKeepDecision(importanceScore, status);
  
  // Determine the reason for the decision
  const reason = determineReason(importanceScore, status, name, durationMs);
  
  // Create and return the sampling decision
  const result: JSON.Obj = new JSON.Obj();
  result.set("importance", importanceScore);
  result.set("keep", keepDecision);
  result.set("reason", reason);
  
  return result.toString();
}

/**
 * Calculate the importance score of the telemetry
 */
function calculateImportance(
  name: string, 
  status: string, 
  kind: string, 
  durationMs: f64,
  attributes: Map<string, string>,
  resourceAttributes: Map<string, string>
): f64 {
  // Start with a base score
  let score: f64 = 0.5;
  
  // Increase score for error statuses
  if (status.includes("Error") || status.includes("error")) {
    score += 0.3;
  }
  
  // Check span name for important patterns
  const lowerName = name.toLowerCase();
  for (let i = 0; i < IMPORTANT_SPAN_PATTERNS.length; i++) {
    if (lowerName.includes(IMPORTANT_SPAN_PATTERNS[i])) {
      score += 0.2;
      break;
    }
  }
  
  // Check for critical service
  const serviceName = resourceAttributes.get("service.name") || "";
  for (let i = 0; i < CRITICAL_SERVICES.length; i++) {
    if (serviceName.includes(CRITICAL_SERVICES[i])) {
      score += 0.2;
      break;
    }
  }
  
  // Check for long duration (slow spans)
  if (durationMs > 1000) {
    score += 0.2;
  } else if (durationMs > 500) {
    score += 0.1;
  }
  
  // Check for important attributes
  for (let i = 0; i < IMPORTANT_ATTRIBUTE_KEYS.length; i++) {
    const key = IMPORTANT_ATTRIBUTE_KEYS[i];
    if (attributes.has(key)) {
      score += 0.1;
    }
  }
  
  // Check for high HTTP status codes
  const httpStatus = attributes.get("http.status_code");
  if (httpStatus) {
    const statusCode = parseInt(httpStatus) as i32;
    if (statusCode >= 500) {
      score += 0.3;
    } else if (statusCode >= 400) {
      score += 0.2;
    }
  }
  
  // Cap the score at 1.0
  if (score > 1.0) {
    score = 1.0;
  }
  
  return score;
}

/**
 * Determine if we should keep this telemetry
 */
function determineKeepDecision(importanceScore: f64, status: string): bool {
  // Always keep errors
  if (status.includes("Error") || status.includes("error")) {
    return true;
  }
  
  // Keep high importance items
  if (importanceScore >= 0.7) {
    return true;
  }
  
  // For medium importance, keep with probability proportional to importance
  if (importanceScore >= 0.4) {
    // Simple deterministic sampling based on importance
    return importanceScore >= 0.5;
  }
  
  // For low importance, sample at 10%
  return importanceScore > 0.9;
}

/**
 * Determine the reason for the sampling decision
 */
function determineReason(importanceScore: f64, status: string, name: string, durationMs: f64): string {
  if (status.includes("Error") || status.includes("error")) {
    return "error_status";
  }
  
  if (durationMs > 1000) {
    return "slow_duration";
  }
  
  const lowerName = name.toLowerCase();
  for (let i = 0; i < IMPORTANT_SPAN_PATTERNS.length; i++) {
    if (lowerName.includes(IMPORTANT_SPAN_PATTERNS[i])) {
      return "important_operation";
    }
  }
  
  if (importanceScore >= 0.7) {
    return "high_importance_score";
  }
  
  return "normal_sampling";
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
 * Helper function to extract a number value from a JSON object
 */
function getNumberValue(obj: JSON.Obj, key: string): f64 {
  const value = obj.get(key);
  if (value) {
    if (value.isInteger) {
      const intVal = value as JSON.Integer;
      return <f64>intVal.valueOf();
    } else if (value.isFloat) {
      const floatVal = value as JSON.Float;
      return floatVal.valueOf();
    }
  }
  return 0;
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