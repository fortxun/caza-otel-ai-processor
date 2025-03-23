package processor

import (
	"context"
	"testing"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor/tests"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata"
	"go.uber.org/zap"
)

func TestTracesProcessor_ProcessTraces_PassThrough(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with all features disabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	traces := testData.CreateTestTraces(nil, nil, pdata.StatusCodeOk)

	// Process the traces
	ctx := context.Background()
	processedTraces, err := processor.processTraces(ctx, traces)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, traces.ResourceSpans().Len(), processedTraces.ResourceSpans().Len())
	
	// Verify no models were called with features disabled
	assert.False(t, mockRuntime.ClassifyErrorCalled)
	assert.False(t, mockRuntime.SampleTelemetryCalled)
	assert.False(t, mockRuntime.ExtractEntitiesCalled)
}

func TestTracesProcessor_ProcessTraces_ErrorClassification(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with error classification enabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       false,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
		Output: OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data with an error span
	testData := &tests.TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "user-service"},
		map[string]interface{}{
			"db.system": "postgresql",
			"db.statement": "SELECT * FROM users WHERE id = ?",
		},
		pdata.StatusCodeError,
	)

	// Process the traces
	ctx := context.Background()
	processedTraces, err := processor.processTraces(ctx, traces)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, traces.ResourceSpans().Len(), processedTraces.ResourceSpans().Len())
	
	// Verify error classifier was called
	assert.True(t, mockRuntime.ClassifyErrorCalled)
	
	// Verify the result contains AI-generated attributes
	rs := processedTraces.ResourceSpans().At(0)
	ils := rs.InstrumentationLibrarySpans().At(0)
	span := ils.Spans().At(0)
	
	// Check for the expected classification attributes
	val, ok := span.Attributes().Get("ai.category")
	assert.True(t, ok)
	assert.Equal(t, "database_error", val.StringVal())
	
	val, ok = span.Attributes().Get("ai.system")
	assert.True(t, ok)
	assert.Equal(t, "postgres", val.StringVal())
	
	val, ok = span.Attributes().Get("ai.owner")
	assert.True(t, ok)
	assert.Equal(t, "database-team", val.StringVal())
}

func TestTracesProcessor_ProcessTraces_EntityExtraction(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with entity extraction enabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    true,
			ContextLinking:      false,
		},
		Output: OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "api-gateway"},
		map[string]interface{}{
			"http.method": "POST",
			"http.url": "/api/users",
		},
		pdata.StatusCodeOk,
	)

	// Process the traces
	ctx := context.Background()
	processedTraces, err := processor.processTraces(ctx, traces)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, traces.ResourceSpans().Len(), processedTraces.ResourceSpans().Len())
	
	// Verify entity extractor was called
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
	
	// Verify the result contains AI-generated attributes
	rs := processedTraces.ResourceSpans().At(0)
	ils := rs.InstrumentationLibrarySpans().At(0)
	span := ils.Spans().At(0)
	
	// Check for expected attributes
	// Note: In a real test, we'd need to handle the JSON array structure,
	// but for simplicity we're just checking if the attributes exist
	_, ok := span.Attributes().Get("ai.services")
	assert.True(t, ok)
	
	_, ok = span.Attributes().Get("ai.dependencies")
	assert.True(t, ok)
	
	_, ok = span.Attributes().Get("ai.operations")
	assert.True(t, ok)
}

func TestTracesProcessor_ProcessTraces_SmartSampling(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with smart sampling enabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       true,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
		Sampling: SamplingConfig{
			ErrorEvents: 1.0,
			SlowSpans:   1.0,
			NormalSpans: 0.5,
			ThresholdMs: 500,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.75,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "payment-service"},
		map[string]interface{}{
			"http.method": "POST",
			"http.url": "/api/payments",
		},
		pdata.StatusCodeOk,
	)

	// Process the traces
	ctx := context.Background()
	processedTraces, err := processor.processTraces(ctx, traces)

	// Verify
	require.NoError(t, err)
	
	// Verify sampler was called
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	
	// With our mock always returning keep=true, we should have the same number of spans
	assert.Equal(t, traces.ResourceSpans().Len(), processedTraces.ResourceSpans().Len())
	
	// Now test with a mock that says don't keep the span
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.2,
		"keep": false,
		"reason": "low_importance",
	}
	
	// Process the traces again
	processedTraces, err = processor.processTraces(ctx, traces)
	
	// Verify
	require.NoError(t, err)
	
	// With our mock now returning keep=false, we should have empty processed traces
	assert.Equal(t, 0, processedTraces.ResourceSpans().Len())
}

func TestTracesProcessor_ProcessTraces_AllFeaturesEnabled(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with all features enabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    true,
			ContextLinking:      true,
		},
		Sampling: SamplingConfig{
			ErrorEvents: 1.0,
			SlowSpans:   1.0,
			NormalSpans: 0.5,
			ThresholdMs: 500,
		},
		Output: OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response to always keep spans
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.9,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data with an error span
	testData := &tests.TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "order-service"},
		map[string]interface{}{
			"db.system": "postgresql",
			"db.statement": "INSERT INTO orders VALUES (?)",
		},
		pdata.StatusCodeError,
	)

	// Process the traces
	ctx := context.Background()
	processedTraces, err := processor.processTraces(ctx, traces)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 1, processedTraces.ResourceSpans().Len())
	
	// Verify all models were called
	assert.True(t, mockRuntime.ClassifyErrorCalled)
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
	
	// Verify the result contains AI-generated attributes
	rs := processedTraces.ResourceSpans().At(0)
	ils := rs.InstrumentationLibrarySpans().At(0)
	span := ils.Spans().At(0)
	
	// Check for error classification
	val, ok := span.Attributes().Get("ai.category")
	assert.True(t, ok)
	assert.Equal(t, "database_error", val.StringVal())
	
	// Check for entity extraction
	_, ok = span.Attributes().Get("ai.services")
	assert.True(t, ok)
}

func TestTracesProcessor_Shutdown(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration
	config := &Config{}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Call shutdown
	ctx := context.Background()
	err := processor.shutdown(ctx)

	// Verify
	require.NoError(t, err)
	assert.True(t, mockRuntime.CloseCalled)
}

func TestNewTracesProcessor(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a minimal configuration for testing
	config := &Config{
		Models: ModelsConfig{
			ErrorClassifier: ModelConfig{
				Path:        "",  // Empty path for testing
				MemoryLimit: 100,
				TimeoutMs:   50,
			},
			ImportanceSampler: ModelConfig{
				Path:        "",  // Empty path for testing
				MemoryLimit: 80,
				TimeoutMs:   30,
			},
			EntityExtractor: ModelConfig{
				Path:        "",  // Empty path for testing
				MemoryLimit: 150,
				TimeoutMs:   50,
			},
		},
		Processing: ProcessingConfig{
			BatchSize:   50,
			Concurrency: 4,
			QueueSize:   1000,
			TimeoutMs:   500,
		},
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
		Sampling: SamplingConfig{
			ErrorEvents: 1.0,
			SlowSpans:   1.0,
			NormalSpans: 0.1,
			ThresholdMs: 500,
		},
		Output: OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockTracesConsumer{}

	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}

	// Create a new traces processor
	processor, err := newTracesProcessor(logger, config, nextConsumer)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}