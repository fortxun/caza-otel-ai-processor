package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// TestWasmIntegration tests the full integration of the processor with actual WASM models
func TestWasmIntegration(t *testing.T) {
	// Skip if not fullwasm build
	if os.Getenv("FULLWASM_TEST") != "1" {
		t.Skip("Skipping WASM integration test - requires FULLWASM_TEST=1 environment variable")
	}

	// Find WASM models - they should be in ../../../wasm-models/*/build/*.wasm
	repoRoot := filepath.Join("..", "..", "..")
	errorClassifierPath := filepath.Join(repoRoot, "wasm-models", "error-classifier", "build", "error-classifier.wasm")
	importanceSamplerPath := filepath.Join(repoRoot, "wasm-models", "importance-sampler", "build", "importance-sampler.wasm")
	entityExtractorPath := filepath.Join(repoRoot, "wasm-models", "entity-extractor", "build", "entity-extractor.wasm")

	// Verify WASM models exist
	if _, err := os.Stat(errorClassifierPath); os.IsNotExist(err) {
		t.Fatalf("Error classifier WASM model not found at %s", errorClassifierPath)
	}
	if _, err := os.Stat(importanceSamplerPath); os.IsNotExist(err) {
		t.Fatalf("Importance sampler WASM model not found at %s", importanceSamplerPath)
	}
	if _, err := os.Stat(entityExtractorPath); os.IsNotExist(err) {
		t.Fatalf("Entity extractor WASM model not found at %s", entityExtractorPath)
	}

	// Create a logger
	logger, _ := zap.NewDevelopment()
	logger.Info("Starting WASM integration test", 
		zap.String("error_classifier", errorClassifierPath),
		zap.String("importance_sampler", importanceSamplerPath),
		zap.String("entity_extractor", entityExtractorPath))

	// Create a factory
	factory := processor.NewFactory()

	// Get the default config
	defaultConfig := factory.CreateDefaultConfig().(*processor.Config)
	
	// Configure WASM models
	defaultConfig.Models.ErrorClassifier.Path = errorClassifierPath
	defaultConfig.Models.ErrorClassifier.MemoryLimitMB = 100
	defaultConfig.Models.ErrorClassifier.TimeoutMs = 50

	defaultConfig.Models.ImportanceSampler.Path = importanceSamplerPath
	defaultConfig.Models.ImportanceSampler.MemoryLimitMB = 80
	defaultConfig.Models.ImportanceSampler.TimeoutMs = 30

	defaultConfig.Models.EntityExtractor.Path = entityExtractorPath
	defaultConfig.Models.EntityExtractor.MemoryLimitMB = 150
	defaultConfig.Models.EntityExtractor.TimeoutMs = 50

	// Enable all features for testing
	defaultConfig.Features.ErrorClassification = true
	defaultConfig.Features.SmartSampling = true
	defaultConfig.Features.EntityExtraction = true

	// Configure sampling settings
	defaultConfig.Sampling.ErrorEvents = 1.0
	defaultConfig.Sampling.SlowSpans = 1.0
	defaultConfig.Sampling.NormalSpans = 0.5
	defaultConfig.Sampling.ThresholdMs = 500

	// Configure processing
	defaultConfig.Processing.BatchSize = 10
	defaultConfig.Processing.Concurrency = 2
	defaultConfig.Processing.QueueSize = 100
	defaultConfig.Processing.TimeoutMs = 250

	// Configure output
	defaultConfig.Output.AttributeNamespace = "ai."
	defaultConfig.Output.IncludeConfidenceScores = true
	defaultConfig.Output.MaxAttributeLength = 256

	// Create processor settings
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{},
		BuildInfo:         component.BuildInfo{},
		Logger:            logger,
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

	// Create test trace data with error
	traces := createTestTraces(
		"database-service",
		"Database connection failed: connection refused to postgres://db:5432",
		ptrace.StatusCodeError,
		600, // Slow span
	)

	// Process the traces
	startTime := time.Now()
	err = tracesProcessor.ConsumeTraces(context.Background(), traces)
	processingDuration := time.Since(startTime)
	
	require.NoError(t, err)
	logger.Info("Trace processing completed", zap.Duration("duration", processingDuration))

	// Verify the consumer received the processed traces
	require.Len(t, tracesConsumer.ConsumedTraces, 1)
	
	// Validate AI attributes were added
	resourceSpans := tracesConsumer.ConsumedTraces[0].ResourceSpans()
	require.Equal(t, 1, resourceSpans.Len())
	
	scopeSpans := resourceSpans.At(0).ScopeSpans()
	require.Equal(t, 1, scopeSpans.Len())
	
	spans := scopeSpans.At(0).Spans()
	require.Equal(t, 1, spans.Len())
	
	span := spans.At(0)
	
	// Check AI attributes
	hasErrorCategory := false
	hasImportance := false
	hasEntityData := false
	
	attributes := span.Attributes()
	attributes.Range(func(k string, v pcommon.Value) bool {
		if k == "ai.error.category" {
			hasErrorCategory = true
			assert.Equal(t, "database_error", v.Str())
		}
		if k == "ai.sampling.importance" {
			hasImportance = true
			assert.True(t, v.Double() > 0)
		}
		if k == "ai.entities.system" {
			hasEntityData = true
		}
		return true
	})
	
	assert.True(t, hasErrorCategory, "Error category attribute not found")
	assert.True(t, hasImportance, "Importance attribute not found")
	assert.True(t, hasEntityData, "Entity data attribute not found")

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
	metrics := createTestMetrics("api-service", "api.latency", 750.0)

	// Process the metrics
	startTime = time.Now()
	err = metricsProcessor.ConsumeMetrics(context.Background(), metrics)
	processingDuration = time.Since(startTime)
	
	require.NoError(t, err)
	logger.Info("Metrics processing completed", zap.Duration("duration", processingDuration))

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

	// Create test log data with error
	logs := createTestLogs(
		"auth-service",
		plog.SeverityNumberError,
		"Authentication failed: invalid token format for user abc123",
	)

	// Process the logs
	startTime = time.Now()
	err = logsProcessor.ConsumeLogs(context.Background(), logs)
	processingDuration = time.Since(startTime)
	
	require.NoError(t, err)
	logger.Info("Logs processing completed", zap.Duration("duration", processingDuration))

	// Verify the consumer received the processed logs
	require.Len(t, logsConsumer.ConsumedLogs, 1)

	// =========================================================================
	// Test Parallel Processing with Traces
	// =========================================================================
	
	// Create a parallel consumer 
	parallelTracesConsumer := &MockTracesConsumer{}
	
	// Create another processor
	parallelProcessor, err := factory.CreateTracesProcessor(
		context.Background(),
		settings,
		defaultConfig,
		parallelTracesConsumer,
	)
	require.NoError(t, err)
	
	// Create a large batch of traces (100 spans)
	manyTraces := createManyTestTraces("users-service", 100)
	
	// Process in parallel
	startTime = time.Now()
	err = parallelProcessor.ConsumeTraces(context.Background(), manyTraces)
	parallelDuration := time.Since(startTime)
	
	require.NoError(t, err)
	logger.Info("Parallel trace processing completed", 
		zap.Duration("duration", parallelDuration),
		zap.Int("numSpans", 100))
	
	// Verify all traces were processed
	require.Len(t, parallelTracesConsumer.ConsumedTraces, 1)
	totalSpans := 0
	rss := parallelTracesConsumer.ConsumedTraces[0].ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		ss := rss.At(i).ScopeSpans()
		for j := 0; j < ss.Len(); j++ {
			totalSpans += ss.At(j).Spans().Len()
		}
	}
	require.Equal(t, 100, totalSpans, "Expected 100 spans to be processed")

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
	
	err = parallelProcessor.Shutdown(context.Background())
	require.NoError(t, err)
}

// Helper test data creation functions

// createTestTraces creates a single trace for testing
func createTestTraces(serviceName string, errorMessage string, statusCode ptrace.StatusCode, durationMs int64) ptrace.Traces {
	traces := ptrace.NewTraces()
	
	rs := traces.ResourceSpans().AppendEmpty()
	resource := rs.Resource()
	resource.Attributes().PutStr("service.name", serviceName)
	resource.Attributes().PutStr("deployment.environment", "production")
	
	ss := rs.ScopeSpans().AppendEmpty()
	scope := ss.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	span := ss.Spans().AppendEmpty()
	span.SetName("database.query")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	
	startTime := pcommon.Timestamp(100000000)
	span.SetStartTimestamp(startTime)
	span.SetEndTimestamp(startTime + pcommon.Timestamp(durationMs*1000000)) // Convert ms to ns
	
	span.Status().SetCode(statusCode)
	if errorMessage != "" {
		span.Status().SetMessage(errorMessage)
	}
	
	// Add some attributes
	span.Attributes().PutStr("db.system", "postgresql")
	span.Attributes().PutStr("db.statement", "SELECT * FROM users WHERE id = ?")
	span.Attributes().PutStr("db.operation", "query")
	span.Attributes().PutBool("internal", true)
	span.Attributes().PutInt("retry_count", 0)
	
	// Add an error event if status is error
	if statusCode == ptrace.StatusCodeError {
		event := span.Events().AppendEmpty()
		event.SetName("exception")
		event.Attributes().PutStr("exception.type", "ConnectionError")
		event.Attributes().PutStr("exception.message", errorMessage)
		event.Attributes().PutBool("exception.escaped", true)
	}
	
	return traces
}

// createManyTestTraces creates multiple traces for testing parallel processing
func createManyTestTraces(serviceName string, count int) ptrace.Traces {
	traces := ptrace.NewTraces()
	
	rs := traces.ResourceSpans().AppendEmpty()
	resource := rs.Resource()
	resource.Attributes().PutStr("service.name", serviceName)
	resource.Attributes().PutStr("deployment.environment", "production")
	
	ss := rs.ScopeSpans().AppendEmpty()
	scope := ss.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	for i := 0; i < count; i++ {
		span := ss.Spans().AppendEmpty()
		
		// Randomize span types for more realistic test
		if i % 5 == 0 {
			// Every 5th span is an error
			span.SetName("error.operation")
			span.Status().SetCode(ptrace.StatusCodeError)
			span.Status().SetMessage("Error in operation")
			
			event := span.Events().AppendEmpty()
			event.SetName("exception")
			event.Attributes().PutStr("exception.message", "Something went wrong")
		} else if i % 3 == 0 {
			// Every 3rd span is slow
			span.SetName("slow.operation")
			span.Status().SetCode(ptrace.StatusCodeOk)
			// Set duration to 600ms (slow)
			span.SetStartTimestamp(pcommon.Timestamp(100000000))
			span.SetEndTimestamp(pcommon.Timestamp(100000000 + 600*1000000))
		} else {
			// Normal span
			span.SetName("normal.operation")
			span.Status().SetCode(ptrace.StatusCodeOk)
			// Set duration to 50ms (normal)
			span.SetStartTimestamp(pcommon.Timestamp(100000000))
			span.SetEndTimestamp(pcommon.Timestamp(100000000 + 50*1000000))
		}
		
		// Set IDs
		span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, byte(i % 256), 14, 15, 16}))
		span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, byte(i % 256)}))
		
		// Add some attributes
		span.Attributes().PutStr("operation.type", "test")
		span.Attributes().PutInt("operation.index", int64(i))
	}
	
	return traces
}

// createTestMetrics creates a metric for testing
func createTestMetrics(serviceName string, metricName string, value float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	resource := rm.Resource()
	resource.Attributes().PutStr("service.name", serviceName)
	resource.Attributes().PutStr("deployment.environment", "production")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	scope := sm.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(metricName)
	metric.SetDescription("Test metric for " + metricName)
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.Timestamp(100000000))
	dp.SetDoubleValue(value)
	
	// Add some attributes
	dp.Attributes().PutStr("operation", "api_call")
	dp.Attributes().PutStr("endpoint", "/api/users")
	dp.Attributes().PutBool("internal", false)
	dp.Attributes().PutInt("instance_id", 1)
	
	return metrics
}

// createTestLogs creates a log for testing
func createTestLogs(serviceName string, severity plog.SeverityNumber, message string) plog.Logs {
	logs := plog.NewLogs()
	
	rl := logs.ResourceLogs().AppendEmpty()
	resource := rl.Resource()
	resource.Attributes().PutStr("service.name", serviceName)
	resource.Attributes().PutStr("deployment.environment", "production")
	
	sl := rl.ScopeLogs().AppendEmpty()
	scope := sl.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	log := sl.LogRecords().AppendEmpty()
	log.SetTimestamp(pcommon.Timestamp(100000000))
	log.Body().SetStr(message)
	
	log.SetSeverityNumber(severity)
	if severity == plog.SeverityNumberError {
		log.SetSeverityText("ERROR")
	} else if severity == plog.SeverityNumberWarn {
		log.SetSeverityText("WARN")
	} else {
		log.SetSeverityText("INFO")
	}
	
	// Add some attributes
	log.Attributes().PutStr("component", "authentication")
	log.Attributes().PutStr("user_id", "abc123")
	log.Attributes().PutBool("internal", true)
	log.Attributes().PutInt("attempt", 3)
	
	return logs
}

// Mock consumers for testing
type MockTracesConsumer struct {
	ConsumedTraces []ptrace.Traces
}

func (m *MockTracesConsumer) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	m.ConsumedTraces = append(m.ConsumedTraces, td)
	return nil
}

func (m *MockTracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type MockMetricsConsumer struct {
	ConsumedMetrics []pmetric.Metrics
}

func (m *MockMetricsConsumer) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	m.ConsumedMetrics = append(m.ConsumedMetrics, md)
	return nil
}

func (m *MockMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type MockLogsConsumer struct {
	ConsumedLogs []plog.Logs
}

func (m *MockLogsConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	m.ConsumedLogs = append(m.ConsumedLogs, ld)
	return nil
}

func (m *MockLogsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}