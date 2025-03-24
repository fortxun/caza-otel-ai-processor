package tests

import (
	"context"
	"testing"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	otprocessor "go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// Note: Tests are skipped in CI via TestMain in test_main.go

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
	assert.Equal(t, "ai_processor", factory.Type().String())

	// Get the default config
	defaultConfig := factory.CreateDefaultConfig()
	assert.NotNil(t, defaultConfig)
	assert.IsType(t, &processor.Config{}, defaultConfig)

	// Mock the WASM runtime
	var originalNewWasmRuntime = runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()

	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return NewMockWasmRuntime(), nil
	}

	// Create processor settings
	settings := otprocessor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: logger,
		},
	}

	// =========================================================================
	// Test Traces Processor
	// =========================================================================

	// Create a mock consumer
	tracesConsumer := &MockTracesConsumer{}

	// Create the traces processor
	tracesProcessor, err := factory.CreateTraces(
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
		ptrace.StatusCodeError,
	)

	// Process the traces
	err = tracesProcessor.ConsumeTraces(context.Background(), traces)
	require.NoError(t, err)

	// Verify the consumer received the processed traces
	require.NotEmpty(t, tracesConsumer.ConsumedTraces)

	// =========================================================================
	// Test Metrics Processor
	// =========================================================================

	// Create a mock consumer
	metricsConsumer := &MockMetricsConsumer{}

	// Create the metrics processor
	metricsProcessor, err := factory.CreateMetrics(
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
	require.NotEmpty(t, metricsConsumer.ConsumedMetrics)

	// =========================================================================
	// Test Logs Processor
	// =========================================================================

	// Create a mock consumer
	logsConsumer := &MockLogsConsumer{}

	// Create the logs processor
	logsProcessor, err := factory.CreateLogs(
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
		plog.SeverityNumberError,
		"Database connection error: connection refused to postgres://orders-db:5432",
	)

	// Process the logs
	err = logsProcessor.ConsumeLogs(context.Background(), logs)
	require.NoError(t, err)

	// Verify the consumer received the processed logs
	require.NotEmpty(t, logsConsumer.ConsumedLogs)

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