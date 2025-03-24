//go:build fullwasm
// +build fullwasm

// This file contains the full WASM runtime implementation using wasmer-go
// Only built when using the fullwasm build tag

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	wasmer "github.com/wasmerio/wasmer-go/wasmer"
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
	errorClassifier  *wasmer.Instance
	sampler          *wasmer.Instance
	entityExtractor  *wasmer.Instance
	mutex            sync.RWMutex
	
	// Caches for model results
	errorClassifierCache *ModelResultsCache
	samplerCache         *ModelResultsCache
	entityExtractorCache *ModelResultsCache
	
	// Function overrides for testing
	ClassifyErrorFunc    func(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error)
	SampleTelemetryFunc  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ExtractEntitiesFunc  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	CloseFunc            func() error
}

// NewWasmRuntime creates a new WASM runtime and loads the models.
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

	// Load error classifier model if path is specified
	if config.ErrorClassifierPath != "" {
		instance, err := loadWasmModel(config.ErrorClassifierPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load error classifier model: %w", err)
		}
		runtime.errorClassifier = instance
		logger.Info("Loaded error classifier model", zap.String("path", config.ErrorClassifierPath))
	}

	// Load sampler model if path is specified
	if config.SamplerPath != "" {
		instance, err := loadWasmModel(config.SamplerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load sampler model: %w", err)
		}
		runtime.sampler = instance
		logger.Info("Loaded sampler model", zap.String("path", config.SamplerPath))
	}

	// Load entity extractor model if path is specified
	if config.EntityExtractorPath != "" {
		instance, err := loadWasmModel(config.EntityExtractorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load entity extractor model: %w", err)
		}
		runtime.entityExtractor = instance
		logger.Info("Loaded entity extractor model", zap.String("path", config.EntityExtractorPath))
	}

	return runtime, nil
}

// ClassifyError classifies an error using the error classifier model.
func (r *WasmRuntime) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if r.ClassifyErrorFunc != nil {
		return r.ClassifyErrorFunc(ctx, errorInfo)
	}

	// Check cache first if enabled
	if r.errorClassifierCache != nil {
		if cachedResult, found := r.errorClassifierCache.Get(errorInfo); found {
			return cachedResult, nil
		}
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.errorClassifier == nil {
		return nil, fmt.Errorf("error classifier model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(errorInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error info: %w", err)
	}

	// Call the WASM function
	result, err := r.invokeWasmFunction(r.errorClassifier, "classify_error", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke error classifier: %w", err)
	}

	// Parse the result
	var classification map[string]interface{}
	if err := json.Unmarshal([]byte(result), &classification); err != nil {
		return nil, fmt.Errorf("failed to unmarshal classification result: %w", err)
	}

	// Cache the result if caching is enabled
	if r.errorClassifierCache != nil {
		r.errorClassifierCache.Put(errorInfo, classification)
	}

	return classification, nil
}

// SampleTelemetry determines whether to sample a telemetry item.
func (r *WasmRuntime) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if r.SampleTelemetryFunc != nil {
		return r.SampleTelemetryFunc(ctx, telemetryItem)
	}

	// Check cache first if enabled
	if r.samplerCache != nil {
		if cachedResult, found := r.samplerCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.sampler == nil {
		return nil, fmt.Errorf("sampler model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(telemetryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telemetry item: %w", err)
	}

	// Call the WASM function
	result, err := r.invokeWasmFunction(r.sampler, "sample_telemetry", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke sampler: %w", err)
	}

	// Parse the result
	var samplingDecision map[string]interface{}
	if err := json.Unmarshal([]byte(result), &samplingDecision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sampling decision: %w", err)
	}

	// Cache the result if caching is enabled
	if r.samplerCache != nil {
		r.samplerCache.Put(telemetryItem, samplingDecision)
	}

	return samplingDecision, nil
}

// ExtractEntities extracts entities from a telemetry item.
func (r *WasmRuntime) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if r.ExtractEntitiesFunc != nil {
		return r.ExtractEntitiesFunc(ctx, telemetryItem)
	}

	// Check cache first if enabled
	if r.entityExtractorCache != nil {
		if cachedResult, found := r.entityExtractorCache.Get(telemetryItem); found {
			return cachedResult, nil
		}
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.entityExtractor == nil {
		return nil, fmt.Errorf("entity extractor model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(telemetryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telemetry item: %w", err)
	}

	// Call the WASM function
	result, err := r.invokeWasmFunction(r.entityExtractor, "extract_entities", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke entity extractor: %w", err)
	}

	// Parse the result
	var entities map[string]interface{}
	if err := json.Unmarshal([]byte(result), &entities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entities: %w", err)
	}

	// Cache the result if caching is enabled
	if r.entityExtractorCache != nil {
		r.entityExtractorCache.Put(telemetryItem, entities)
	}

	return entities, nil
}

// ReloadModel reloads a specific model.
func (r *WasmRuntime) ReloadModel(modelType string, path string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	instance, err := loadWasmModel(path)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}

	switch modelType {
	case "error_classifier":
		if r.errorClassifier != nil {
			r.errorClassifier.Close()
		}
		r.errorClassifier = instance
	case "sampler":
		if r.sampler != nil {
			r.sampler.Close()
		}
		r.sampler = instance
	case "entity_extractor":
		if r.entityExtractor != nil {
			r.entityExtractor.Close()
		}
		r.entityExtractor = instance
	default:
		return fmt.Errorf("unknown model type: %s", modelType)
	}

	r.logger.Info("Reloaded model", zap.String("type", modelType), zap.String("path", path))
	return nil
}

// Close cleans up resources used by the WASM runtime.
func (r *WasmRuntime) Close() error {
	// If we have a testing override, use it
	if r.CloseFunc != nil {
		return r.CloseFunc()
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.errorClassifier != nil {
		r.errorClassifier.Close()
		r.errorClassifier = nil
	}

	if r.sampler != nil {
		r.sampler.Close()
		r.sampler = nil
	}

	if r.entityExtractor != nil {
		r.entityExtractor.Close()
		r.entityExtractor = nil
	}

	return nil
}

// Helper functions

// loadWasmModel loads a WASM model from a file.
func loadWasmModel(path string) (*wasmer.Instance, error) {
	// Read the WASM file
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	// Create a new WebAssembly Store
	store := wasmer.NewStore(wasmer.NewEngine())

	// Compile the WASM module
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	// Create import object with required functions for AssemblyScript
	importObject := wasmer.NewImportObject()
	
	// Create required functions for AssemblyScript
	// The WASM module requires env.abort function
	abortFn := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), 
			wasmer.NewValueTypes(),
		),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			// Log the abort information
			msgPtr := args[0].I32()
			filePtr := args[1].I32()
			line := args[2].I32()
			col := args[3].I32()
			fmt.Printf("AssemblyScript abort called: msg=%d file=%d line=%d col=%d\n", 
				msgPtr, filePtr, line, col)
			return []wasmer.Value{}, nil
		},
	)
	
	// Register the required imports
	importObject.Register(
		"env", 
		map[string]wasmer.IntoExtern{
			"abort": abortFn,
		},
	)

	// Instantiate the WASM module
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	return instance, nil
}

// invokeWasmFunction invokes a function in a WASM instance.
func (r *WasmRuntime) invokeWasmFunction(instance *wasmer.Instance, functionName, input string) (string, error) {
	// Log that we're invoking a WASM function
	r.logger.Debug("Invoking WASM function",
		zap.String("function", functionName),
		zap.String("input_sample", input[:min(len(input), 50)]+"..."),
	)

	// Get the function from the instance
	function, err := instance.Exports.GetFunction(functionName)
	if err != nil {
		return "", fmt.Errorf("function %s not found: %w", functionName, err)
	}

	// Invoke the function with the input
	result, err := function(input)
	if err != nil {
		return "", fmt.Errorf("failed to invoke function %s: %w", functionName, err)
	}

	// Convert the result to a string
	resultStr, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result type from function %s", functionName)
	}

	// Log the result
	r.logger.Debug("WASM function returned result",
		zap.String("function", functionName),
		zap.String("result_sample", resultStr[:min(len(resultStr), 50)]+"..."),
	)

	return resultStr, nil
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}