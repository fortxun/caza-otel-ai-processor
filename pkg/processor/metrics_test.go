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

func TestMetricsProcessor_ProcessMetrics_PassThrough(t *testing.T) {
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
	nextConsumer := &tests.MockMetricsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &metricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	metrics := testData.CreateTestMetrics(nil, "", 42.0)

	// Process the metrics
	ctx := context.Background()
	processedMetrics, err := processor.processMetrics(ctx, metrics)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, metrics.ResourceMetrics().Len(), processedMetrics.ResourceMetrics().Len())
	
	// Verify no models were called with features disabled
	assert.False(t, mockRuntime.ClassifyErrorCalled)
	assert.False(t, mockRuntime.SampleTelemetryCalled)
	assert.False(t, mockRuntime.ExtractEntitiesCalled)
}

func TestMetricsProcessor_ProcessMetrics_EntityExtraction(t *testing.T) {
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
	nextConsumer := &tests.MockMetricsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &metricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	metrics := testData.CreateTestMetrics(
		map[string]interface{}{"service.name": "api-gateway"},
		"api.request.duration",
		142.5,
	)

	// Process the metrics
	ctx := context.Background()
	processedMetrics, err := processor.processMetrics(ctx, metrics)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, metrics.ResourceMetrics().Len(), processedMetrics.ResourceMetrics().Len())
	
	// Verify entity extractor was called
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
	
	// Verify the result contains AI-generated attributes
	// The implementation details for checking metrics attributes depend on how you've 
	// implemented the metrics data points access, but the pattern is similar to traces
}

func TestMetricsProcessor_ProcessMetrics_SmartSampling(t *testing.T) {
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
	nextConsumer := &tests.MockMetricsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.75,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &metricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	metrics := testData.CreateTestMetrics(
		map[string]interface{}{"service.name": "payment-service"},
		"payment.latency",
		550.0, // High value that might be important
	)

	// Process the metrics
	ctx := context.Background()
	processedMetrics, err := processor.processMetrics(ctx, metrics)

	// Verify
	require.NoError(t, err)
	
	// Verify sampler was called
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	
	// With our mock always returning keep=true, we should have the same number of metrics
	assert.Equal(t, metrics.ResourceMetrics().Len(), processedMetrics.ResourceMetrics().Len())
	
	// Now test with a mock that says don't keep the metric
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.2,
		"keep": false,
		"reason": "low_importance",
	}
	
	// Process the metrics again
	processedMetrics, err = processor.processMetrics(ctx, metrics)
	
	// Verify
	require.NoError(t, err)
	
	// With our mock now returning keep=false, we should have empty processed metrics
	assert.Equal(t, 0, processedMetrics.ResourceMetrics().Len())
}

func TestMetricsProcessor_ProcessMetrics_AllFeaturesEnabled(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with all applicable features enabled
	// Note: Error classification doesn't apply to metrics, so it's disabled
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: false,
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
	nextConsumer := &tests.MockMetricsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response to always keep metrics
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.9,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &metricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	metrics := testData.CreateTestMetrics(
		map[string]interface{}{"service.name": "order-service"},
		"order.processing.time",
		750.0,
	)

	// Process the metrics
	ctx := context.Background()
	processedMetrics, err := processor.processMetrics(ctx, metrics)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 1, processedMetrics.ResourceMetrics().Len())
	
	// Verify models were called
	assert.False(t, mockRuntime.ClassifyErrorCalled) // Not applicable for metrics
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
}

func TestMetricsProcessor_Shutdown(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration
	config := &Config{}

	// Create a mock consumer
	nextConsumer := &tests.MockMetricsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &metricsProcessor{
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

func TestNewMetricsProcessor(t *testing.T) {
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
	nextConsumer := &tests.MockMetricsConsumer{}

	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}

	// Create a new metrics processor
	processor, err := newMetricsProcessor(logger, config, nextConsumer)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}