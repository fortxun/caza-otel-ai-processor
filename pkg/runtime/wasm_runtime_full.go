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

	"go.uber.org/zap"
	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

// fullWasmImpl is the implementation of wasmRuntimeImpl for the full WASM version
type fullWasmImpl struct {
	logger           *zap.Logger
	errorClassifier  *wasmer.Instance
	sampler          *wasmer.Instance
	entityExtractor  *wasmer.Instance
	
	// Function overrides for testing
	ClassifyErrorFunc    func(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error)
	SampleTelemetryFunc  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ExtractEntitiesFunc  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	CloseFunc            func() error
}

// NewWasmRuntime creates a new WASM runtime and loads the models.
func NewWasmRuntime(logger *zap.Logger, config *WasmRuntimeConfig) (*WasmRuntime, error) {
	// Initialize the common runtime components
	runtime, err := initializeRuntime(logger, config)
	if err != nil {
		return nil, err
	}

	// Create the full WASM implementation
	impl := &fullWasmImpl{
		logger: logger,
	}

	// Load error classifier model if path is specified
	if config.ErrorClassifierPath != "" {
		instance, err := loadWasmModel(config.ErrorClassifierPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load error classifier model: %w", err)
		}
		impl.errorClassifier = instance
		logger.Info("Loaded error classifier model", zap.String("path", config.ErrorClassifierPath))
	}

	// Load sampler model if path is specified
	if config.SamplerPath != "" {
		instance, err := loadWasmModel(config.SamplerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load sampler model: %w", err)
		}
		impl.sampler = instance
		logger.Info("Loaded sampler model", zap.String("path", config.SamplerPath))
	}

	// Load entity extractor model if path is specified
	if config.EntityExtractorPath != "" {
		instance, err := loadWasmModel(config.EntityExtractorPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load entity extractor model: %w", err)
		}
		impl.entityExtractor = instance
		logger.Info("Loaded entity extractor model", zap.String("path", config.EntityExtractorPath))
	}

	// Set the implementation
	runtime.impl = impl

	return runtime, nil
}

// ClassifyError classifies an error using the error classifier model.
func (f *fullWasmImpl) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if f.ClassifyErrorFunc != nil {
		return f.ClassifyErrorFunc(ctx, errorInfo)
	}

	if f.errorClassifier == nil {
		return nil, fmt.Errorf("error classifier model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(errorInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error info: %w", err)
	}

	// Call the WASM function
	result, err := f.invokeWasmFunction(f.errorClassifier, "classify_error", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke error classifier: %w", err)
	}

	// Parse the result
	var classification map[string]interface{}
	if err := json.Unmarshal([]byte(result), &classification); err != nil {
		return nil, fmt.Errorf("failed to unmarshal classification result: %w", err)
	}

	return classification, nil
}

// SampleTelemetry determines whether to sample a telemetry item.
func (f *fullWasmImpl) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if f.SampleTelemetryFunc != nil {
		return f.SampleTelemetryFunc(ctx, telemetryItem)
	}

	if f.sampler == nil {
		return nil, fmt.Errorf("sampler model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(telemetryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telemetry item: %w", err)
	}

	// Call the WASM function
	result, err := f.invokeWasmFunction(f.sampler, "sample_telemetry", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke sampler: %w", err)
	}

	// Parse the result
	var samplingDecision map[string]interface{}
	if err := json.Unmarshal([]byte(result), &samplingDecision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sampling decision: %w", err)
	}

	return samplingDecision, nil
}

// ExtractEntities extracts entities from a telemetry item.
func (f *fullWasmImpl) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	// If we have a testing override, use it
	if f.ExtractEntitiesFunc != nil {
		return f.ExtractEntitiesFunc(ctx, telemetryItem)
	}

	if f.entityExtractor == nil {
		return nil, fmt.Errorf("entity extractor model not loaded")
	}

	// Convert input to JSON
	input, err := json.Marshal(telemetryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telemetry item: %w", err)
	}

	// Call the WASM function
	result, err := f.invokeWasmFunction(f.entityExtractor, "extract_entities", string(input))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke entity extractor: %w", err)
	}

	// Parse the result
	var entities map[string]interface{}
	if err := json.Unmarshal([]byte(result), &entities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entities: %w", err)
	}

	return entities, nil
}

// ReloadModel reloads a specific model.
func (f *fullWasmImpl) ReloadModel(modelType string, path string) error {
	instance, err := loadWasmModel(path)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}

	switch modelType {
	case "error_classifier":
		if f.errorClassifier != nil {
			f.errorClassifier.Close()
		}
		f.errorClassifier = instance
	case "sampler":
		if f.sampler != nil {
			f.sampler.Close()
		}
		f.sampler = instance
	case "entity_extractor":
		if f.entityExtractor != nil {
			f.entityExtractor.Close()
		}
		f.entityExtractor = instance
	default:
		return fmt.Errorf("unknown model type: %s", modelType)
	}

	f.logger.Info("Reloaded model", zap.String("type", modelType), zap.String("path", path))
	return nil
}

// Close cleans up resources used by the WASM runtime.
func (f *fullWasmImpl) Close() error {
	// If we have a testing override, use it
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}

	if f.errorClassifier != nil {
		f.errorClassifier.Close()
		f.errorClassifier = nil
	}

	if f.sampler != nil {
		f.sampler.Close()
		f.sampler = nil
	}

	if f.entityExtractor != nil {
		f.entityExtractor.Close()
		f.entityExtractor = nil
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
func (f *fullWasmImpl) invokeWasmFunction(instance *wasmer.Instance, functionName, input string) (string, error) {
	// Log that we're invoking a WASM function
	f.logger.Debug("Invoking WASM function",
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
	f.logger.Debug("WASM function returned result",
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