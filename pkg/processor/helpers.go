// This file contains helper functions for the processor implementations
// that just forward to the common package

package processor

import (
	"github.com/fortxun/caza-otel-ai-processor/pkg/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Helper function wrappers that delegate to the common package

// attributesToMap converts an OpenTelemetry attribute map to a Go map
func attributesToMap(attributes pcommon.Map) map[string]interface{} {
	return common.AttributesToMap(attributes)
}

// calculateAttributeMapHash calculates a hash for an attribute map
func calculateAttributeMapHash(attributes pcommon.Map) uint64 {
	return common.CalculateAttributeMapHash(attributes)
}

// setAttribute sets an attribute in an OpenTelemetry attribute map
func setAttribute(attributes pcommon.Map, key string, value interface{}) {
	common.SetAttribute(attributes, key, value)
}

// resourcesEqual checks if two resources are equal by comparing their hashes
func resourcesEqual(r1, r2 pcommon.Resource) bool {
	return common.ResourcesEqual(r1, r2)
}

// calculateResourceHash calculates a hash for a resource based on its attributes
func calculateResourceHash(r pcommon.Resource) uint64 {
	return common.CalculateResourceHash(r)
}

// randomSample returns true if the sample should be kept
// based on the sampling rate (0.0-1.0)
func randomSample(rate float64) bool {
	return common.RandomSample(rate)
}

// getOrCreateResource finds a matching resource in the traces or creates a new one
func getOrCreateResource(traces ptrace.Traces, resource pcommon.Resource) ptrace.ResourceSpans {
	return common.GetOrCreateTraceResource(traces, resource)
}

// getOrCreateScope finds a matching scope in the resource spans or creates a new one
func getOrCreateScope(rs ptrace.ResourceSpans, scope pcommon.InstrumentationScope) ptrace.ScopeSpans {
	return common.GetOrCreateScope(rs, scope)
}