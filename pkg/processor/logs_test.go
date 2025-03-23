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

func TestLogsProcessor_ProcessLogs_PassThrough(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(nil, pdata.SeverityNumberInfo, "")

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, logs.ResourceLogs().Len(), processedLogs.ResourceLogs().Len())
	
	// Verify no models were called with features disabled
	assert.False(t, mockRuntime.ClassifyErrorCalled)
	assert.False(t, mockRuntime.SampleTelemetryCalled)
	assert.False(t, mockRuntime.ExtractEntitiesCalled)
}

func TestLogsProcessor_ProcessLogs_ErrorClassification(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data with an error log
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "user-service"},
		pdata.SeverityNumberError,
		"Failed to connect to database: connection refused",
	)

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, logs.ResourceLogs().Len(), processedLogs.ResourceLogs().Len())
	
	// Verify error classifier was called
	assert.True(t, mockRuntime.ClassifyErrorCalled)
	
	// Verify the result contains AI-generated attributes
	rl := processedLogs.ResourceLogs().At(0)
	ill := rl.InstrumentationLibraryLogs().At(0)
	log := ill.Logs().At(0)
	
	// Check for the expected classification attributes
	val, ok := log.Attributes().Get("ai.category")
	assert.True(t, ok)
	assert.Equal(t, "database_error", val.StringVal())
	
	val, ok = log.Attributes().Get("ai.system")
	assert.True(t, ok)
	assert.Equal(t, "postgres", val.StringVal())
	
	val, ok = log.Attributes().Get("ai.owner")
	assert.True(t, ok)
	assert.Equal(t, "database-team", val.StringVal())
}

func TestLogsProcessor_ProcessLogs_EntityExtraction(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "api-gateway"},
		pdata.SeverityNumberInfo,
		"User login successful for user_id=123456 from client=mobile-app",
	)

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, logs.ResourceLogs().Len(), processedLogs.ResourceLogs().Len())
	
	// Verify entity extractor was called
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
	
	// Verify the result contains AI-generated attributes
	rl := processedLogs.ResourceLogs().At(0)
	ill := rl.InstrumentationLibraryLogs().At(0)
	log := ill.Logs().At(0)
	
	// Check for expected attributes
	// Note: In a real test, we'd need to handle the JSON array structure,
	// but for simplicity we're just checking if the attributes exist
	_, ok := log.Attributes().Get("ai.services")
	assert.True(t, ok)
	
	_, ok = log.Attributes().Get("ai.dependencies")
	assert.True(t, ok)
	
	_, ok = log.Attributes().Get("ai.operations")
	assert.True(t, ok)
}

func TestLogsProcessor_ProcessLogs_SmartSampling(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.75,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "payment-service"},
		pdata.SeverityNumberWarn,
		"Payment processing delayed for order #12345",
	)

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	
	// Verify sampler was called
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	
	// With our mock always returning keep=true, we should have the same number of logs
	assert.Equal(t, logs.ResourceLogs().Len(), processedLogs.ResourceLogs().Len())
	
	// Now test with a mock that says don't keep the log
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.2,
		"keep": false,
		"reason": "low_importance",
	}
	
	// Process the logs again
	processedLogs, err = processor.processLogs(ctx, logs)
	
	// Verify
	require.NoError(t, err)
	
	// With our mock now returning keep=false, we should have empty processed logs
	assert.Equal(t, 0, processedLogs.ResourceLogs().Len())
}

func TestLogsProcessor_ProcessLogs_ErrorLogs_AlwaysKept(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration with smart sampling enabled and configured to keep all errors
	config := &Config{
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       true,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
		Sampling: SamplingConfig{
			ErrorEvents: 1.0,  // Always keep errors
			SlowSpans:   1.0,
			NormalSpans: 0.5,
			ThresholdMs: 500,
		},
	}

	// Create a mock consumer
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.1,  // Low importance, would normally be dropped
		"keep": false,
		"reason": "low_importance",
	}

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data with an error log
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "user-service"},
		pdata.SeverityNumberError,  // This is an error log
		"Failed to authenticate user: invalid credentials",
	)

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	
	// Even though the sampler says to drop it, error logs should always be kept
	assert.Equal(t, logs.ResourceLogs().Len(), processedLogs.ResourceLogs().Len())
}

func TestLogsProcessor_ProcessLogs_AllFeaturesEnabled(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()
	
	// Set up the mock sampler response to always keep logs
	mockRuntime.SampleTelemetryOutput = map[string]interface{}{
		"importance": 0.9,
		"keep": true,
		"reason": "high_importance_score",
	}

	// Create the processor
	processor := &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  mockRuntime,
	}

	// Create test data with an error log
	testData := &tests.TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "order-service"},
		pdata.SeverityNumberError,
		"Database connection error: connection refused to postgres://orders-db:5432",
	)

	// Process the logs
	ctx := context.Background()
	processedLogs, err := processor.processLogs(ctx, logs)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 1, processedLogs.ResourceLogs().Len())
	
	// Verify all models were called
	assert.True(t, mockRuntime.ClassifyErrorCalled)
	assert.True(t, mockRuntime.SampleTelemetryCalled)
	assert.True(t, mockRuntime.ExtractEntitiesCalled)
	
	// Verify the result contains AI-generated attributes
	rl := processedLogs.ResourceLogs().At(0)
	ill := rl.InstrumentationLibraryLogs().At(0)
	log := ill.Logs().At(0)
	
	// Check for error classification
	val, ok := log.Attributes().Get("ai.category")
	assert.True(t, ok)
	assert.Equal(t, "database_error", val.StringVal())
	
	// Check for entity extraction
	_, ok = log.Attributes().Get("ai.services")
	assert.True(t, ok)
}

func TestLogsProcessor_Shutdown(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a configuration
	config := &Config{}

	// Create a mock consumer
	nextConsumer := &tests.MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := tests.NewMockWasmRuntime()

	// Create the processor
	processor := &logsProcessor{
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

func TestNewLogsProcessor(t *testing.T) {
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
	nextConsumer := &tests.MockLogsConsumer{}

	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}

	// Create a new logs processor
	processor, err := newLogsProcessor(logger, config, nextConsumer)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}