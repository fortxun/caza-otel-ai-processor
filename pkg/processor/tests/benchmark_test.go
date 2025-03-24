package tests

import (
	"context"
	"testing"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// BenchmarkTracesProcessor_PassThrough benchmarks the traces processor with all features disabled
func BenchmarkTracesProcessor_PassThrough(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration with all features disabled
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
	}

	// Create a mock consumer
	nextConsumer := &MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create the processor
	proc := &processor.TracesProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nextConsumer,
		WasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &TestData{}
	traces := testData.CreateTestTraces(nil, nil, ptrace.StatusCodeOk)

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessTraces(ctx, traces)
	}
}

// BenchmarkTracesProcessor_WithFeatures benchmarks the traces processor with features enabled
func BenchmarkTracesProcessor_WithFeatures(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration with features enabled
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    true,
			ContextLinking:      false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &MockTracesConsumer{}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create the processor
	proc := &processor.TracesProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nextConsumer,
		WasmRuntime:  mockRuntime,
	}

	// Create test data with error spans for classification
	testData := &TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "user-service"},
		map[string]interface{}{
			"db.system": "postgresql",
			"db.statement": "SELECT * FROM users WHERE id = ?",
		},
		ptrace.StatusCodeError,
	)

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessTraces(ctx, traces)
	}
}

// BenchmarkLogsProcessor_PassThrough benchmarks the logs processor with all features disabled
func BenchmarkLogsProcessor_PassThrough(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration with all features disabled
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
	}

	// Create a mock consumer
	nextConsumer := &MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create the processor
	proc := &processor.LogsProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nextConsumer,
		WasmRuntime:  mockRuntime,
	}

	// Create test data
	testData := &TestData{}
	logs := testData.CreateTestLogs(nil, plog.SeverityNumberInfo, "")

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessLogs(ctx, logs)
	}
}

// BenchmarkLogsProcessor_WithFeatures benchmarks the logs processor with features enabled
func BenchmarkLogsProcessor_WithFeatures(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration with features enabled
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    true,
			ContextLinking:      false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create a mock consumer
	nextConsumer := &MockLogsConsumer{}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create the processor
	proc := &processor.LogsProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nextConsumer,
		WasmRuntime:  mockRuntime,
	}

	// Create test data with error logs
	testData := &TestData{}
	logs := testData.CreateTestLogs(
		map[string]interface{}{"service.name": "user-service"},
		plog.SeverityNumberError,
		"Failed to connect to database: connection refused",
	)

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessLogs(ctx, logs)
	}
}

// BenchmarkSamplingDecision benchmarks the sampling decision logic
func BenchmarkSamplingDecision(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			SmartSampling: true,
		},
		Sampling: processor.SamplingConfig{
			ErrorEvents: 1.0,
			SlowSpans:   1.0,
			NormalSpans: 0.1,
			ThresholdMs: 500,
		},
	}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create test data
	testData := &TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "payment-service"},
		map[string]interface{}{
			"http.method": "POST",
			"http.url": "/api/payments",
		},
		ptrace.StatusCodeOk,
	)

	// Extract a resource and span for sampling
	rs := traces.ResourceSpans().At(0)
	ils := rs.InstrumentationLibrarySpans().At(0)
	span := ils.Spans().At(0)

	// Create the trace processor since it has the sampling logic
	proc := &processor.TracesProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nil,
		WasmRuntime:  mockRuntime,
	}

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = proc.MakeSamplingDecision(ctx, span, rs.Resource())
	}
}

// BenchmarkErrorClassification benchmarks the error classification logic
func BenchmarkErrorClassification(b *testing.B) {
	// Create a test logger
	logger := zap.NewNop()

	// Create a configuration
	config := &processor.Config{
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
		},
	}

	// Create a mock WASM runtime
	mockRuntime := NewMockWasmRuntime()

	// Create test data with an error span
	testData := &TestData{}
	traces := testData.CreateTestTraces(
		map[string]interface{}{"service.name": "user-service"},
		map[string]interface{}{
			"db.system": "postgresql",
			"db.statement": "SELECT * FROM users WHERE id = ?",
		},
		ptrace.StatusCodeError,
	)

	// Extract a resource and span for classification
	rs := traces.ResourceSpans().At(0)
	ils := rs.InstrumentationLibrarySpans().At(0)
	span := ils.Spans().At(0)

	// Create the trace processor since it has the classification logic
	proc := &processor.TracesProcessor{
		Logger:       logger,
		Config:       config,
		NextConsumer: nil,
		WasmRuntime:  mockRuntime,
	}

	// Run the benchmark
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proc.ClassifyError(ctx, span, rs.Resource())
	}
}

// Mock processors for benchmarking

type TracesProcessor struct {
	Logger       *zap.Logger
	Config       *processor.Config
	NextConsumer *MockTracesConsumer
	WasmRuntime  *MockWasmRuntime
}

func (p *TracesProcessor) ProcessTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	return td, nil
}

func (p *TracesProcessor) MakeSamplingDecision(ctx context.Context, span ptrace.Span, resource pcommon.Resource) bool {
	return true
}

func (p *TracesProcessor) ClassifyError(ctx context.Context, span ptrace.Span, resource pcommon.Resource) {
	// No-op
}

type LogsProcessor struct {
	Logger       *zap.Logger
	Config       *processor.Config
	NextConsumer *MockLogsConsumer
	WasmRuntime  *MockWasmRuntime
}

func (p *LogsProcessor) ProcessLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	return ld, nil
}