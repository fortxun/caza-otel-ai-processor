//go:build !fullwasm
// +build !fullwasm

// This file contains a stub implementation of the WASM runtime
// Used when building without the fullwasm tag

package runtime

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// stubImpl is the implementation of wasmRuntimeImpl for the stub version
type stubImpl struct {
	logger *zap.Logger
}

// NewWasmRuntime creates a new WASM runtime and loads the models.
// This is a stubbed version that logs model paths but doesn't actually load WASM modules
func NewWasmRuntime(logger *zap.Logger, config *WasmRuntimeConfig) (*WasmRuntime, error) {
	// Initialize the common runtime components
	runtime, err := initializeRuntime(logger, config)
	if err != nil {
		return nil, err
	}

	// Create the stub implementation
	stubImpl := &stubImpl{
		logger: logger,
	}

	// Set the implementation
	runtime.impl = stubImpl
	
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
func (s *stubImpl) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	s.logger.Info("Stub ClassifyError called", zap.Any("errorInfo", errorInfo))
	
	// Return stub classification
	classification := map[string]interface{}{
		"error_type": "unknown",
		"error_cause": "system",
		"severity": "medium", 
		"confidence": 0.95,
		"owner": "platform-team",
	}
	
	return classification, nil
}

// SampleTelemetry determines whether to sample a telemetry item.
// In the stub version, it returns a default sampling decision
func (s *stubImpl) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	s.logger.Info("Stub SampleTelemetry called", zap.Any("telemetryItem", telemetryItem))
	
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
	if name != "" && (len(name) > 3 && (name[:3] == "db." || name[:3] == "sql")) {
		importance = 0.8 // higher importance for database operations
	}

	// Return stub sampling decision
	result := map[string]interface{}{
		"importance": importance,
		"keep": importance > 0.3,
		"confidence": 0.9,
	}
	
	return result, nil
}

// ExtractEntities extracts entities from a telemetry item.
// In the stub version, it returns default entities based on telemetry attributes
func (s *stubImpl) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	s.logger.Info("Stub ExtractEntities called", zap.Any("telemetryItem", telemetryItem))
	
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
	
	return entities, nil
}

// ReloadModel reloads a specific model.
// In the stub version, it just logs the reload
func (s *stubImpl) ReloadModel(modelType string, path string) error {
	s.logger.Info("Stub ReloadModel called", 
		zap.String("type", modelType), 
		zap.String("path", path))
	return nil
}

// Close cleans up resources used by the WASM runtime.
// In the stub version, it just logs the close
func (s *stubImpl) Close() error {
	s.logger.Info("Stub Close called")
	return nil
}