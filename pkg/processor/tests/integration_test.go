package tests

import (
	"context"
	"testing"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata"
	"go.uber.org/zap"
)

// TestProcessorIntegration tests the full integration of the processor
func TestProcessorIntegration(t *testing.T) {
	// Skip in CI since we can't load the actual WASM models
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a factory
	factory := processor.NewFactory()

	// Validate factory name
	assert.Equal(t, "ai_processor", factory.Type())

	// Get the default config
	defaultConfig := factory.CreateDefaultConfig()
	assert.NotNil(t, defaultConfig)
	assert.IsType(t, &processor.Config{}, defaultConfig)

	// Mock the WASM runtime
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()

	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return createMockWasmRuntime(logger), nil
	}

	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger:               logger,
		MetricsLevel:         component.MetricsLevelDetailed,
		MetricsReporter:      nil,
		BuildInfo:            component.BuildInfo{},
		TelemetrySettings:    component.TelemetrySettings{},
		ExtensionBundle:      nil,
		AttributesProcessor:  nil,
		ServiceExtensions:    nil,
		DisableLoggerBackend: false,
		TracerProvider:       nil,
		MeterProvider:        nil,
	}

	// =========================================================================
	// Test Traces Processor
	// =========================================================================

	// Create a mock consumer
	tracesConsumer := &MockTracesConsumer{}

	// Create the traces processor
	tracesProcessor, err := factory.CreateTracesProcessor(
		context.Background(),
		settings,
		defaultConfig,
		tracesConsumer,
	)
	require.NoError(t, err)
	require.NotNil(t, tracesProcessor)

	// Create test trace data
	testData := &TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "user-service"},
		map[string]interface{}{
			"db.system": "postgresql",
			"db.statement": "SELECT * FROM users WHERE id = ?",
		},
		pdata.StatusCodeError,
	)

	// Process the traces
	err = tracesProcessor.ConsumeTraces(context.Background(), traces)
	require.NoError(t, err)

	// Verify the consumer received the processed traces
	require.Len(t, tracesConsumer.ConsumedTraces, 1)

	// =========================================================================
	// Test Metrics Processor
	// =========================================================================

	// Create a mock consumer
	metricsConsumer := &MockMetricsConsumer{}

	// Create the metrics processor
	metricsProcessor, err := factory.CreateMetricsProcessor(
		context.Background(),
		settings,
		defaultConfig,
		metricsConsumer,
	)
	require.NoError(t, err)
	require.NotNil(t, metricsProcessor)

	// Create test metric data
	metrics := testData.CreateTestMetrics(
		map[string]interface{}{"service.name": "payment-service"},
		"payment.latency",
		550.0,
	)

	// Process the metrics
	err = metricsProcessor.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Verify the consumer received the processed metrics
	require.Len(t, metricsConsumer.ConsumedMetrics, 1)

	// =========================================================================
	// Test Logs Processor
	// =========================================================================

	// Create a mock consumer
	logsConsumer := &MockLogsConsumer{}

	// Create the logs processor
	logsProcessor, err := factory.CreateLogsProcessor(
		context.Background(),
		settings,
		defaultConfig,
		logsConsumer,
	)
	require.NoError(t, err)
	require.NotNil(t, logsProcessor)

	// Create test log data
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "order-service"},
		pdata.SeverityNumberError,
		"Database connection error: connection refused to postgres://orders-db:5432",
	)

	// Process the logs
	err = logsProcessor.ConsumeLogs(context.Background(), logs)
	require.NoError(t, err)

	// Verify the consumer received the processed logs
	require.Len(t, logsConsumer.ConsumedLogs, 1)

	// =========================================================================
	// Test Shutdown
	// =========================================================================

	// Shutdown the processors
	err = tracesProcessor.Shutdown(context.Background())
	require.NoError(t, err)

	err = metricsProcessor.Shutdown(context.Background())
	require.NoError(t, err)

	err = logsProcessor.Shutdown(context.Background())
	require.NoError(t, err)
}

// createMockWasmRuntime creates a mock WASM runtime for testing
func createMockWasmRuntime(logger *zap.Logger) *runtime.WasmRuntime {
	// Create a mock runtime that returns predictable results
	mockRuntime := &runtime.WasmRuntime{}

	// Mock the methods
	mockRuntime.ClassifyError = func(ctx context.Context, errorInfo map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"category":   "database_error",
			"system":     "postgres",
			"owner":      "database-team",
			"severity":   "high",
			"impact":     "medium",
			"confidence": 0.85,
		}, nil
	}

	mockRuntime.SampleTelemetry = func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"importance": 0.75,
			"keep":       true,
			"reason":     "high_importance_score",
		}, nil
	}

	mockRuntime.ExtractEntities = func(ctx context.Context, telemetryItem map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"services":     []string{"user-service", "api-gateway"},
			"dependencies": []string{"postgres", "redis"},
			"operations":   []string{"get_user", "update_account"},
			"confidence":   0.82,
		}, nil
	}

	mockRuntime.Close = func() error {
		return nil
	}

	return mockRuntime
}