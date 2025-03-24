// This file defines the interfaces and shared types for the WASM runtime
// It's used by both the stub and full implementations

package runtime

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// WasmRuntimeConfig defines the configuration for the Wasm runtime.
type WasmRuntimeConfig struct {
	ErrorClassifierPath   string
	ErrorClassifierMemory int
	SamplerPath           string
	SamplerMemory         int
	EntityExtractorPath   string
	EntityExtractorMemory int
	
	// EnableModelCaching enables caching model results
	EnableModelCaching bool
	
	// ModelCacheSize defines the size of the model results cache
	ModelCacheSize int
	
	// ModelCacheTTLSeconds defines the TTL for cached model results
	ModelCacheTTLSeconds int
}

// WasmRuntime manages the WASM modules and provides methods to invoke them.
type WasmRuntime struct {
	logger           *zap.Logger
	mutex            sync.RWMutex
	
	// Caches for model results
	errorClassifierCache *ModelResultsCache
	samplerCache         *ModelResultsCache
	entityExtractorCache *ModelResultsCache
	
	// Implementation details are in the implementation-specific files
	impl wasmRuntimeImpl
}

// Interface for the implementation-specific parts
type wasmRuntimeImpl interface {
	ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error)
	SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ReloadModel(modelType string, path string) error
	Close() error
}

// Public API methods that delegate to the implementation

// ClassifyError classifies an error using the error classifier model.
func (r *WasmRuntime) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	// Check cache first if enabled
	if r.errorClassifierCache != nil {
		if cachedResult, found := r.errorClassifierCache.Get(errorInfo); found {
			return cachedResult, nil
		}
	}

	// Call the implementation
	result, err := r.impl.ClassifyError(ctx, errorInfo)
	if err != nil {
		return nil, err
	}

	// Cache the result if caching is enabled
	if r.errorClassifierCache != nil {
		r.errorClassifierCache.Put(errorInfo, result)
	}

	return result, nil
}

// SampleTelemetry determines whether to sample a telemetry item.
func (r *WasmRuntime) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// Check cache first if enabled
	if r.samplerCache != nil {
		if cachedResult, found := r.samplerCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	// Call the implementation
	result, err := r.impl.SampleTelemetry(ctx, telemetryItem)
	if err != nil {
		return nil, err
	}

	// Cache the result if caching is enabled
	if r.samplerCache != nil {
		r.samplerCache.Put(telemetryItem, result)
	}

	return result, nil
}

// ExtractEntities extracts entities from a telemetry item.
func (r *WasmRuntime) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// Check cache first if enabled
	if r.entityExtractorCache != nil {
		if cachedResult, found := r.entityExtractorCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	// Call the implementation
	result, err := r.impl.ExtractEntities(ctx, telemetryItem)
	if err != nil {
		return nil, err
	}

	// Cache the result if caching is enabled
	if r.entityExtractorCache != nil {
		r.entityExtractorCache.Put(telemetryItem, result)
	}

	return result, nil
}

// ReloadModel reloads a specific model.
func (r *WasmRuntime) ReloadModel(modelType string, path string) error {
	return r.impl.ReloadModel(modelType, path)
}

// Close cleans up resources used by the WASM runtime.
func (r *WasmRuntime) Close() error {
	return r.impl.Close()
}

// Helper function to initialize the runtime
func initializeRuntime(logger *zap.Logger, config *WasmRuntimeConfig) (*WasmRuntime, error) {
	runtime := &WasmRuntime{
		logger: logger,
		mutex:  sync.RWMutex{},
	}
	
	// Initialize caches if enabled
	if config.EnableModelCaching {
		// Default TTL to 60 seconds if not specified
		ttl := config.ModelCacheTTLSeconds
		if ttl == 0 {
			ttl = 60
		}
		
		// Create caches for each model
		var err error
		
		// Error classifier cache
		runtime.errorClassifierCache, err = NewModelResultsCache(config.ModelCacheSize, ttl)
		if err != nil {
			return nil, err
		}
		
		// Sampler cache
		runtime.samplerCache, err = NewModelResultsCache(config.ModelCacheSize, ttl)
		if err != nil {
			return nil, err
		}
		
		// Entity extractor cache
		runtime.entityExtractorCache, err = NewModelResultsCache(config.ModelCacheSize, ttl)
		if err != nil {
			return nil, err
		}
		
		logger.Info("Enabled model result caching",
			zap.Int("cache_size", config.ModelCacheSize),
			zap.Int("ttl_seconds", ttl))
	}

	return runtime, nil
}