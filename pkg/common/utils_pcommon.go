// Package common provides shared utilities for both full and stub implementations
package common

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// AttributesToMap converts an OpenTelemetry attribute map to a Go map
func AttributesToMap(attributes pcommon.Map) map[string]interface{} {
	// If the attribute map is empty, return an empty map
	if attributes.Len() == 0 {
		return make(map[string]interface{})
	}

	// Calculate a hash for the attribute map to use as a cache key
	hash := CalculateAttributeMapHash(attributes)
	
	// Check if we have the map in cache
	if cachedMap, found := AttributeMapCache.Load(hash); found {
		// Return a copy of the cached map to avoid concurrent modification
		result := make(map[string]interface{}, len(cachedMap.(map[string]interface{})))
		for k, v := range cachedMap.(map[string]interface{}) {
			result[k] = v
		}
		return result
	}
	
	// Not in cache, convert the map
	result := make(map[string]interface{}, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		switch v.Type() {
		case pcommon.ValueTypeStr:
			result[k] = v.Str()
		case pcommon.ValueTypeBool:
			result[k] = v.Bool()
		case pcommon.ValueTypeInt:
			result[k] = v.Int()
		case pcommon.ValueTypeDouble:
			result[k] = v.Double()
		}
		return true
	})
	
	// Store in cache
	AttributeMapCache.Store(hash, result)
	
	return result
}

// SetAttribute sets an attribute in an OpenTelemetry attribute map
func SetAttribute(attributes pcommon.Map, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		attributes.PutStr(key, v)
	case bool:
		attributes.PutBool(key, v)
	case int:
		attributes.PutInt(key, int64(v))
	case int64:
		attributes.PutInt(key, v)
	case float64:
		attributes.PutDouble(key, v)
	}
}

// ResourcesEqual checks if two resources are equal by comparing their attributes
func ResourcesEqual(r1, r2 pcommon.Resource) bool {
	// Fast path: pointer equality
	if &r1 == &r2 {
		return true
	}
	
	// Get or calculate hash for r1
	hash1, ok := ResourceCache.Load(r1)
	if !ok {
		hash1 = CalculateResourceHash(r1)
		ResourceCache.Store(r1, hash1)
	}
	
	// Get or calculate hash for r2
	hash2, ok := ResourceCache.Load(r2)
	if !ok {
		hash2 = CalculateResourceHash(r2)
		ResourceCache.Store(r2, hash2)
	}
	
	// Compare hashes
	return hash1 == hash2
}

// SamplerRand is a global random number generator for sampling
var SamplerRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var SamplerMutex sync.Mutex

// RandomSample returns true if the sample should be kept
// based on the sampling rate (0.0-1.0)
func RandomSample(rate float64) bool {
	// Fast path for common cases
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}
	
	// Get a thread-safe random number between 0.0 and 1.0
	SamplerMutex.Lock()
	r := SamplerRand.Float64()
	SamplerMutex.Unlock()
	
	// Keep if random number is less than the rate
	return r < rate
}

// AttributeMapCache is a cache of attribute maps to avoid repeated conversions
// The key is a hash of the attribute map, and the value is the converted map
var AttributeMapCache sync.Map

// CalculateAttributeMapHash calculates a hash for an attribute map
// This is used as a cache key for the AttributesToMap function
func CalculateAttributeMapHash(attributes pcommon.Map) uint64 {
	// Use FNV-1a hash algorithm
	h := uint64(14695981039346656037) // FNV offset basis
	
	// Sort keys for deterministic hashing
	keys := make([]string, 0, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	sort.Strings(keys)
	
	// Hash each key-value pair
	for _, k := range keys {
		// Hash the key
		for i := 0; i < len(k); i++ {
			h ^= uint64(k[i])
			h *= 1099511628211 // FNV prime
		}
		
		// Hash the value
		v, _ := attributes.Get(k)
		switch v.Type() {
		case pcommon.ValueTypeStr:
			s := v.Str()
			for i := 0; i < len(s); i++ {
				h ^= uint64(s[i])
				h *= 1099511628211
			}
		case pcommon.ValueTypeBool:
			if v.Bool() {
				h ^= 1
			} else {
				h ^= 0
			}
			h *= 1099511628211
		case pcommon.ValueTypeInt:
			val := v.Int()
			h ^= uint64(val)
			h *= 1099511628211
		case pcommon.ValueTypeDouble:
			val := v.Double()
			h ^= uint64(math.Float64bits(val))
			h *= 1099511628211
		}
	}
	
	return h
}

// GetOrCreateTraceResource finds a matching resource in the traces or creates a new one
func GetOrCreateTraceResource(traces ptrace.Traces, resource pcommon.Resource) ptrace.ResourceSpans {
	rss := traces.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		// Check if this resource matches
		if ResourcesEqual(rs.Resource(), resource) {
			return rs
		}
	}
	
	// Resource not found, create a new one
	rs := traces.ResourceSpans().AppendEmpty()
	resource.CopyTo(rs.Resource())
	return rs
}

// GetOrCreateScope finds a matching scope in the resource spans or creates a new one
func GetOrCreateScope(rs ptrace.ResourceSpans, scope pcommon.InstrumentationScope) ptrace.ScopeSpans {
	sss := rs.ScopeSpans()
	for i := 0; i < sss.Len(); i++ {
		ss := sss.At(i)
		// Check if this scope matches
		if ss.Scope().Name() == scope.Name() && ss.Scope().Version() == scope.Version() {
			return ss
		}
	}
	
	// Scope not found, create a new one
	ss := rs.ScopeSpans().AppendEmpty()
	scope.CopyTo(ss.Scope())
	return ss
}

// CalculateResourceHash calculates a hash for a resource based on its attributes
func CalculateResourceHash(r pcommon.Resource) uint64 {
	return CalculateAttributeMapHash(r.Attributes())
}

// ResourceCache is a cache of resource hashes for fast comparison
var ResourceCache sync.Map