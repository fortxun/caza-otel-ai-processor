package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockWasmerInstance is a mock for testing without actual WASM files
type MockWasmerInstance struct {
	FunctionResults map[string]string
	FunctionErrors  map[string]error
	CloseCalled     bool
}

func (m *MockWasmerInstance) Close() {
	m.CloseCalled = true
}

// TestNewWasmRuntime tests creating a new WASM runtime
func TestNewWasmRuntime(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Mock the loadWasmModel function to avoid needing actual WASM files
	originalLoadWasmModel := loadWasmModel
	defer func() { loadWasmModel = originalLoadWasmModel }()
	
	// Replace with mock implementation
	loadWasmModel = func(path string) (*MockWasmerInstance, error) {
		mock := &MockWasmerInstance{
			FunctionResults: map[string]string{
				"classify_error":   `{"category":"database_error","system":"postgres","owner":"database-team","severity":"high","impact":"medium","confidence":0.85}`,
				"sample_telemetry": `{"importance":0.75,"keep":true,"reason":"high_importance_score"}`,
				"extract_entities": `{"services":["user-service","api-gateway"],"dependencies":["postgres","redis"],"operations":["get_user","update_account"],"confidence":0.82}`,
			},
			FunctionErrors: map[string]error{},
		}
		return mock, nil
	}
	
	// Create a runtime configuration
	config := &WasmRuntimeConfig{
		ErrorClassifierPath:   "/path/to/error-classifier.wasm",
		ErrorClassifierMemory: 100,
		SamplerPath:           "/path/to/sampler.wasm",
		SamplerMemory:         80,
		EntityExtractorPath:   "/path/to/entity-extractor.wasm",
		EntityExtractorMemory: 150,
	}
	
	// Create a new WASM runtime
	runtime, err := NewWasmRuntime(logger, config)
	
	// Verify runtime creation
	assert.NoError(t, err)
	assert.NotNil(t, runtime)
}

// TestClassifyError tests the ClassifyError method
func TestClassifyError(t *testing.T) {
	// Mock runtime for testing
	runtime := createMockRuntime(t)
	
	// Test input
	errorInfo := map[string]interface{}{
		"name":       "ExecuteQuery",
		"status":     "Connection refused to database",
		"kind":       "CLIENT",
		"attributes": map[string]interface{}{
			"db.system": "postgresql",
			"db.name":   "users",
		},
		"resource": map[string]interface{}{
			"service.name": "user-service",
		},
	}
	
	// Call the function
	ctx := context.Background()
	result, err := runtime.ClassifyError(ctx, errorInfo)
	
	// Verify the result
	assert.NoError(t, err)
	assert.Equal(t, "database_error", result["category"])
	assert.Equal(t, "postgres", result["system"])
	assert.Equal(t, "database-team", result["owner"])
}

// TestSampleTelemetry tests the SampleTelemetry method
func TestSampleTelemetry(t *testing.T) {
	// Mock runtime for testing
	runtime := createMockRuntime(t)
	
	// Test input
	telemetryInfo := map[string]interface{}{
		"name":       "ProcessPayment",
		"status":     "OK",
		"kind":       "CLIENT",
		"duration":   150,
		"attributes": map[string]interface{}{
			"http.method": "POST",
			"http.url":    "/api/payments",
		},
		"resource": map[string]interface{}{
			"service.name": "payment-service",
		},
	}
	
	// Call the function
	ctx := context.Background()
	result, err := runtime.SampleTelemetry(ctx, telemetryInfo)
	
	// Verify the result
	assert.NoError(t, err)
	assert.Equal(t, 0.75, result["importance"])
	assert.Equal(t, true, result["keep"])
	assert.Equal(t, "high_importance_score", result["reason"])
}

// TestExtractEntities tests the ExtractEntities method
func TestExtractEntities(t *testing.T) {
	// Mock runtime for testing
	runtime := createMockRuntime(t)
	
	// Test input
	telemetryInfo := map[string]interface{}{
		"name":        "HandleUserRequest",
		"description": "Process user API request",
		"type":        "span",
		"attributes":  map[string]interface{}{
			"http.method": "POST",
			"http.url":    "/api/users",
		},
		"resource": map[string]interface{}{
			"service.name": "user-service",
		},
	}
	
	// Call the function
	ctx := context.Background()
	result, err := runtime.ExtractEntities(ctx, telemetryInfo)
	
	// Verify the result
	assert.NoError(t, err)
	services, ok := result["services"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, services, "user-service")
	
	dependencies, ok := result["dependencies"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, dependencies, "postgres")
	
	operations, ok := result["operations"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, operations, "get_user")
}

// TestReloadModel tests the ReloadModel method
func TestReloadModel(t *testing.T) {
	// Mock runtime for testing
	runtime := createMockRuntime(t)
	
	// Mock the loadWasmModel function
	originalLoadWasmModel := loadWasmModel
	defer func() { loadWasmModel = originalLoadWasmModel }()
	
	loadCount := 0
	loadWasmModel = func(path string) (*MockWasmerInstance, error) {
		loadCount++
		return &MockWasmerInstance{
			FunctionResults: map[string]string{
				"classify_error": `{"category":"network_error","system":"api","owner":"platform-team","severity":"medium","impact":"low","confidence":0.9}`,
			},
		}, nil
	}
	
	// Reload the error classifier model
	err := runtime.ReloadModel("error_classifier", "/path/to/new/error-classifier.wasm")
	
	// Verify the model was reloaded
	assert.NoError(t, err)
	assert.Equal(t, 1, loadCount)
	
	// Test with an unknown model type
	err = runtime.ReloadModel("unknown_model", "/path/to/model.wasm")
	
	// Verify the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model type")
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	// Mock runtime for testing
	runtime := createMockRuntime(t)
	
	// Close the runtime
	err := runtime.Close()
	
	// Verify the runtime was closed
	assert.NoError(t, err)
	
	// Check that all models were closed
	errorClassifier, ok := runtime.errorClassifier.(*MockWasmerInstance)
	assert.True(t, ok)
	assert.True(t, errorClassifier.CloseCalled)
	
	sampler, ok := runtime.sampler.(*MockWasmerInstance)
	assert.True(t, ok)
	assert.True(t, sampler.CloseCalled)
	
	entityExtractor, ok := runtime.entityExtractor.(*MockWasmerInstance)
	assert.True(t, ok)
	assert.True(t, entityExtractor.CloseCalled)
}

// Helper function to create a mock runtime for testing
func createMockRuntime(t *testing.T) *WasmRuntime {
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create mock instances
	errorClassifier := &MockWasmerInstance{
		FunctionResults: map[string]string{
			"classify_error": `{"category":"database_error","system":"postgres","owner":"database-team","severity":"high","impact":"medium","confidence":0.85}`,
		},
	}
	
	sampler := &MockWasmerInstance{
		FunctionResults: map[string]string{
			"sample_telemetry": `{"importance":0.75,"keep":true,"reason":"high_importance_score"}`,
		},
	}
	
	entityExtractor := &MockWasmerInstance{
		FunctionResults: map[string]string{
			"extract_entities": `{"services":["user-service","api-gateway"],"dependencies":["postgres","redis"],"operations":["get_user","update_account"],"confidence":0.82}`,
		},
	}
	
	// Create the runtime
	runtime := &WasmRuntime{
		logger:          logger,
		errorClassifier: errorClassifier,
		sampler:         sampler,
		entityExtractor: entityExtractor,
	}
	
	// Mock the invokeWasmFunction method
	runtime.invokeWasmFunction = func(instance interface{}, functionName string, input string) (string, error) {
		mockInstance, ok := instance.(*MockWasmerInstance)
		assert.True(t, ok)
		
		result, ok := mockInstance.FunctionResults[functionName]
		assert.True(t, ok)
		
		return result, nil
	}
	
	return runtime
}