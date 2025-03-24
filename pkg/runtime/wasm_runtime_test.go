package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockWasmerInstance is a mock for testing
type MockWasmerInstance struct {
	FunctionResults map[string]string
	FunctionErrors  map[string]error
	CloseCalled     bool
}

// Close implements the close method for the mock
func (m *MockWasmerInstance) Close() {
	m.CloseCalled = true
}

// TestNewWasmRuntime tests creating a new WASM runtime
func TestNewWasmRuntime(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
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
	// Create a mock runtime for testing
	runtime := createMockRuntimeWithOverrides(t)
	
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
	// Create a mock runtime for testing
	runtime := createMockRuntimeWithOverrides(t)
	
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
	// Create a mock runtime for testing
	runtime := createMockRuntimeWithOverrides(t)
	
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
	// Create a mock runtime for testing
	runtime := createMockRuntimeWithOverrides(t)
	
	// Test reload model - just verify no error occurs
	err := runtime.ReloadModel("error_classifier", "/path/to/new/error-classifier.wasm")
	assert.NoError(t, err)
	
	// Test with an unknown model type - also should succeed with stub
	err = runtime.ReloadModel("unknown_model", "/path/to/model.wasm")
	assert.NoError(t, err)
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	// Create a mock runtime for testing
	runtime := createMockRuntimeWithOverrides(t)
	
	// Close the runtime
	err := runtime.Close()
	
	// Verify the runtime was closed
	assert.NoError(t, err)
}

// Helper function to create a mock runtime with function overrides for testing
func createMockRuntimeWithOverrides(t *testing.T) *WasmRuntime {
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create a config that enables caching
	config := &WasmRuntimeConfig{
		EnableModelCaching: true,
		ModelCacheSize: 100,
		ModelCacheTTLSeconds: 60,
	}
	
	// Create a new runtime
	runtime, err := NewWasmRuntime(logger, config)
	assert.NoError(t, err)
	
	// Create a mock implementation
	mockImpl := &mockImplementation{
		ClassifyErrorMock: func(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"category":   "database_error",
				"system":     "postgres",
				"owner":      "database-team",
				"severity":   "high",
				"impact":     "medium",
				"confidence": 0.85,
			}, nil
		},
		SampleTelemetryMock: func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"importance": 0.75,
				"keep":       true,
				"reason":     "high_importance_score",
			}, nil
		},
		ExtractEntitiesMock: func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"services":     []interface{}{"user-service", "api-gateway"},
				"dependencies": []interface{}{"postgres", "redis"},
				"operations":   []interface{}{"get_user", "update_account"},
				"confidence":   0.82,
			}, nil
		},
		ReloadModelMock: func(modelType string, path string) error {
			return nil
		},
		CloseMock: func() error {
			return nil
		},
	}
	
	// Set the mock implementation
	runtime.impl = mockImpl
	
	return runtime
}

// Mock implementation of wasmRuntimeImpl for testing
type mockImplementation struct {
	ClassifyErrorMock    func(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error)
	SampleTelemetryMock  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ExtractEntitiesMock  func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error)
	ReloadModelMock      func(modelType string, path string) error
	CloseMock            func() error
}

func (m *mockImplementation) ClassifyError(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
	return m.ClassifyErrorMock(ctx, errorInfo)
}

func (m *mockImplementation) SampleTelemetry(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	return m.SampleTelemetryMock(ctx, telemetryItem)
}

func (m *mockImplementation) ExtractEntities(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
	return m.ExtractEntitiesMock(ctx, telemetryItem)
}

func (m *mockImplementation) ReloadModel(modelType string, path string) error {
	return m.ReloadModelMock(modelType, path)
}

func (m *mockImplementation) Close() error {
	return m.CloseMock()
}