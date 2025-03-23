//go:build !fullwasm
// +build !fullwasm

package runtime

import (
	"context"
	"fmt"
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
// This is a stubbed version to allow building without wasmer-go
type WasmRuntime struct {
	logger           *zap.Logger
	mutex            sync.RWMutex
	
	// Caches for model results
	errorClassifierCache *ModelResultsCache
	samplerCache         *ModelResultsCache
	entityExtractorCache *ModelResultsCache
}

// NewWasmRuntime creates a new WASM runtime and loads the models.
// This is a stubbed version that logs model paths but doesn't actually load WASM modules
func NewWasmRuntime(logger *zap.Logger, config *WasmRuntimeConfig) (*WasmRuntime, error) {
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
			return nil, fmt.Errorf("failed to create error classifier cache: %w", err)
		}
		
		// Sampler cache
		runtime.samplerCache, err = NewModelResultsCache(config.ModelCacheSize, ttl)
		if err != nil {
			return nil, fmt.Errorf("failed to create sampler cache: %w", err)
		}
		
		// Entity extractor cache
		runtime.entityExtractorCache, err = NewModelResultsCache(config.ModelCacheSize, ttl)
		if err != nil {
			return nil, fmt.Errorf("failed to create entity extractor cache: %w", err)
		}
		
		logger.Info("Enabled model result caching",
			zap.Int("cache_size", config.ModelCacheSize),
			zap.Int("ttl_seconds", ttl))
	}

	// Log model paths but don't actually load them in the stub version
	if config.ErrorClassifierPath != "" {
		logger.Info("Would load error classifier model in non-stub version", 
			zap.String("path", config.ErrorClassifierPath))
	}

	if config.SamplerPath != "" {
		logger.Info("Would load sampler model in non-stub version", 
			zap.String("path", config.SamplerPath))
	}

	if config.EntityExtractorPath != "" {
		logger.Info("Would load entity extractor model in non-stub version", 
			zap.String("path", config.EntityExtractorPath))
	}

	return runtime, nil
}

// ClassifyError classifies an error using the error classifier model.
// In the stub version, it returns a default classification
func (r *WasmRuntime) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	r.logger.Info("Stub ClassifyError called", zap.Any("errorInfo", errorInfo))
	
	// Check cache first if enabled
	if r.errorClassifierCache != nil {
		if cachedResult, found := r.errorClassifierCache.Get(errorInfo); found {
			return cachedResult, nil
		}
	}

	// Return stub classification
	classification := map[string]interface{}{
		"error_type": "unknown",
		"error_cause": "system",
		"severity": "medium", 
		"confidence": 0.95,
		"owner": "platform-team",
	}
	
	// Cache the result if caching is enabled
	if r.errorClassifierCache != nil {
		r.errorClassifierCache.Put(errorInfo, classification)
	}

	return classification, nil
}

// SampleTelemetry determines whether to sample a telemetry item.
// In the stub version, it returns a default sampling decision
func (r *WasmRuntime) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	r.logger.Info("Stub SampleTelemetry called", zap.Any("telemetryItem", telemetryItem))
	
	// Check cache first if enabled
	if r.samplerCache != nil {
		if cachedResult, found := r.samplerCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	// Extract some info from the telemetry item to make the stub more realistic
	name, _ := telemetryItem["name"].(string)
	hasError := false
	if status, ok := telemetryItem["status"].(string); ok {
		hasError = status == "error"
	}
	
	// Determine importance based on name and error status
	importance := 0.5 // default medium importance
	if hasError {
		importance = 0.9 // high importance for errors
	}
	if name != "" && (len(name) > 3 && name[:3] == "db." || name[:3] == "sql") {
		importance = 0.8 // higher importance for database operations
	}

	// Return stub sampling decision
	result := map[string]interface{}{
		"importance": importance,
		"keep": importance > 0.3,
		"confidence": 0.9,
	}
	
	// Cache the result if caching is enabled
	if r.samplerCache != nil {
		r.samplerCache.Put(telemetryItem, result)
	}

	return result, nil
}

// ExtractEntities extracts entities from a telemetry item.
// In the stub version, it returns default entities based on telemetry attributes
func (r *WasmRuntime) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	r.logger.Info("Stub ExtractEntities called", zap.Any("telemetryItem", telemetryItem))
	
	// Check cache first if enabled
	if r.entityExtractorCache != nil {
		if cachedResult, found := r.entityExtractorCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	// Extract some info to make stub more realistic
	name, _ := telemetryItem["name"].(string)
	
	// Default entities
	entities := map[string]interface{}{
		"service": "unknown-service",
		"operation_type": "unknown",
		"confidence": 0.8,
	}
	
	// Try to extract more specific entities from the name
	if name != "" {
		if len(name) > 3 && name[:3] == "db." {
			entities["service"] = "database"
			entities["operation_type"] = "data-access"
		} else if len(name) > 4 && name[:4] == "http" {
			entities["service"] = "web-api"
			entities["operation_type"] = "http-request"
		}
	}
	
	// Cache the result if caching is enabled
	if r.entityExtractorCache != nil {
		r.entityExtractorCache.Put(telemetryItem, entities)
	}

	return entities, nil
}

// ReloadModel reloads a specific model.
// In the stub version, it just logs the reload
func (r *WasmRuntime) ReloadModel(modelType string, path string) error {
	r.logger.Info("Stub ReloadModel called", 
		zap.String("type", modelType), 
		zap.String("path", path))
	return nil
}

// Close cleans up resources used by the WASM runtime.
// In the stub version, it just logs the close
func (r *WasmRuntime) Close() error {
	r.logger.Info("Stub Close called")
	return nil
}