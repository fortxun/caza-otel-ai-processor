package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// These benchmarks compare the performance of the WASM implementation vs the stub implementation
// Run with: go test -tags=fullwasm -run=^$ -bench=BenchmarkWasm -benchmem

// BenchmarkWasmVsStub_ErrorClassification compares WASM vs stub for error classification
func BenchmarkWasmVsStub_ErrorClassification(b *testing.B) {
	// Skip if not fullwasm build
	if os.Getenv("FULLWASM_TEST") != "1" {
		b.Skip("Skipping WASM benchmark - requires FULLWASM_TEST=1 environment variable")
	}

	// Find WASM models
	repoRoot := filepath.Join("..", "..", "..")
	errorClassifierPath := filepath.Join(repoRoot, "wasm-models", "error-classifier", "build", "error-classifier.wasm")

	// Verify WASM model exists
	if _, err := os.Stat(errorClassifierPath); os.IsNotExist(err) {
		b.Fatalf("Error classifier WASM model not found at %s", errorClassifierPath)
	}

	// Create loggers
	silentLogger := zap.NewNop()
	
	// Create processor configurations
	fullWasmConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ErrorClassifier: processor.ModelConfig{
				Path:          errorClassifierPath,
				MemoryLimitMB: 100,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       false,
			EntityExtraction:    false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}
	
	stubConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ErrorClassifier: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 100,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       false,
			EntityExtraction:    false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger:            silentLogger,
		BuildInfo:         component.BuildInfo{},
		TelemetrySettings: component.TelemetrySettings{},
	}

	// Test data with error
	traces := createTestTraces(
		"database-service",
		"Database connection failed: connection refused to postgres://db:5432",
		ptrace.StatusCodeError,
		600,
	)

	// Extract a single span for the benchmark
	resourceSpans := traces.ResourceSpans()
	resource := resourceSpans.At(0).Resource()
	span := resourceSpans.At(0).ScopeSpans().At(0).Spans().At(0)

	// Benchmark the WASM implementation
	b.Run("WASM", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			fullWasmConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create WASM processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		wasmRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get WASM runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the error information directly
			errorInfo := map[string]interface{}{
				"status":   span.Status().Message(),
				"name":     span.Name(),
				"attributes": extractAttributes(span.Attributes()),
				"resource":  extractAttributes(resource.Attributes()),
			}

			// Call the WASM function directly for more accurate benchmarking
			_, _ = wasmRuntime.ClassifyError(context.Background(), errorInfo)
		}
	})

	// Benchmark the stub implementation
	b.Run("Stub", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			stubConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create stub processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		stubRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get stub runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the error information directly
			errorInfo := map[string]interface{}{
				"status":   span.Status().Message(),
				"name":     span.Name(),
				"attributes": extractAttributes(span.Attributes()),
				"resource":  extractAttributes(resource.Attributes()),
			}

			// Call the stub function directly for more accurate benchmarking
			_, _ = stubRuntime.ClassifyError(context.Background(), errorInfo)
		}
	})
}

// BenchmarkWasmVsStub_SampleTelemetry compares WASM vs stub for telemetry sampling
func BenchmarkWasmVsStub_SampleTelemetry(b *testing.B) {
	// Skip if not fullwasm build
	if os.Getenv("FULLWASM_TEST") != "1" {
		b.Skip("Skipping WASM benchmark - requires FULLWASM_TEST=1 environment variable")
	}

	// Find WASM models
	repoRoot := filepath.Join("..", "..", "..")
	samplerPath := filepath.Join(repoRoot, "wasm-models", "importance-sampler", "build", "importance-sampler.wasm")

	// Verify WASM model exists
	if _, err := os.Stat(samplerPath); os.IsNotExist(err) {
		b.Fatalf("Importance sampler WASM model not found at %s", samplerPath)
	}

	// Create loggers
	silentLogger := zap.NewNop()
	
	// Create processor configurations
	fullWasmConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ImportanceSampler: processor.ModelConfig{
				Path:          samplerPath,
				MemoryLimitMB: 80,
				TimeoutMs:     30,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       true,
			EntityExtraction:    false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}
	
	stubConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ImportanceSampler: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 80,
				TimeoutMs:     30,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       true,
			EntityExtraction:    false,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger:            silentLogger,
		BuildInfo:         component.BuildInfo{},
		TelemetrySettings: component.TelemetrySettings{},
	}

	// Test data with mixed traces
	traces := createManyTestTraces("users-service", 10)
	
	// Extract a single span for the benchmark
	resourceSpans := traces.ResourceSpans()
	resource := resourceSpans.At(0).Resource()
	span := resourceSpans.At(0).ScopeSpans().At(0).Spans().At(0)

	// Benchmark the WASM implementation
	b.Run("WASM", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			fullWasmConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create WASM processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		wasmRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get WASM runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the telemetry information directly
			telemetryItem := map[string]interface{}{
				"name":      span.Name(),
				"duration":  float64(span.EndTimestamp() - span.StartTimestamp()) / 1e6, // ns to ms
				"status":    int(span.Status().Code()),
				"attributes": extractAttributes(span.Attributes()),
				"resource":   extractAttributes(resource.Attributes()),
			}

			// Call the WASM function directly for more accurate benchmarking
			_, _ = wasmRuntime.SampleTelemetry(context.Background(), telemetryItem)
		}
	})

	// Benchmark the stub implementation
	b.Run("Stub", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			stubConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create stub processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		stubRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get stub runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the telemetry information directly
			telemetryItem := map[string]interface{}{
				"name":      span.Name(),
				"duration":  float64(span.EndTimestamp() - span.StartTimestamp()) / 1e6, // ns to ms
				"status":    int(span.Status().Code()),
				"attributes": extractAttributes(span.Attributes()),
				"resource":   extractAttributes(resource.Attributes()),
			}

			// Call the stub function directly for more accurate benchmarking
			_, _ = stubRuntime.SampleTelemetry(context.Background(), telemetryItem)
		}
	})
}

// BenchmarkWasmVsStub_EntityExtraction compares WASM vs stub for entity extraction
func BenchmarkWasmVsStub_EntityExtraction(b *testing.B) {
	// Skip if not fullwasm build
	if os.Getenv("FULLWASM_TEST") != "1" {
		b.Skip("Skipping WASM benchmark - requires FULLWASM_TEST=1 environment variable")
	}

	// Find WASM models
	repoRoot := filepath.Join("..", "..", "..")
	entityExtractorPath := filepath.Join(repoRoot, "wasm-models", "entity-extractor", "build", "entity-extractor.wasm")

	// Verify WASM model exists
	if _, err := os.Stat(entityExtractorPath); os.IsNotExist(err) {
		b.Fatalf("Entity extractor WASM model not found at %s", entityExtractorPath)
	}

	// Create loggers
	silentLogger := zap.NewNop()
	
	// Create processor configurations
	fullWasmConfig := &processor.Config{
		Models: processor.ModelsConfig{
			EntityExtractor: processor.ModelConfig{
				Path:          entityExtractorPath,
				MemoryLimitMB: 150,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    true,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}
	
	stubConfig := &processor.Config{
		Models: processor.ModelsConfig{
			EntityExtractor: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 150,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       false,
			EntityExtraction:    true,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger:            silentLogger,
		BuildInfo:         component.BuildInfo{},
		TelemetrySettings: component.TelemetrySettings{},
	}

	// Test data with rich information
	traces := createTestTraces(
		"payment-service",
		"Processing payment for user 12345 through provider stripe failed with API error",
		ptrace.StatusCodeError,
		500,
	)
	
	// Add more attributes for entity extraction
	span := traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	span.Attributes().PutStr("http.method", "POST")
	span.Attributes().PutStr("http.url", "/api/payments/process")
	span.Attributes().PutStr("payment.provider", "stripe")
	span.Attributes().PutStr("payment.method", "credit_card")
	span.Attributes().PutStr("user.id", "12345")
	
	// Extract a single span for the benchmark
	resourceSpans := traces.ResourceSpans()
	resource := resourceSpans.At(0).Resource()

	// Benchmark the WASM implementation
	b.Run("WASM", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			fullWasmConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create WASM processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		wasmRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get WASM runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the telemetry information directly
			telemetryItem := map[string]interface{}{
				"name":      span.Name(),
				"status":    span.Status().Message(),
				"attributes": extractAttributes(span.Attributes()),
				"resource":   extractAttributes(resource.Attributes()),
			}

			// Call the WASM function directly for more accurate benchmarking
			_, _ = wasmRuntime.ExtractEntities(context.Background(), telemetryItem)
		}
	})

	// Benchmark the stub implementation
	b.Run("Stub", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			stubConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create stub processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Get the WASM runtime from the processor
		stubRuntime, ok := proc.(*processor.TracesProcessor).GetWasmRuntime().(*runtime.WasmRuntime)
		if !ok {
			b.Fatal("Failed to get stub runtime from processor")
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Extract the telemetry information directly
			telemetryItem := map[string]interface{}{
				"name":      span.Name(),
				"status":    span.Status().Message(),
				"attributes": extractAttributes(span.Attributes()),
				"resource":   extractAttributes(resource.Attributes()),
			}

			// Call the stub function directly for more accurate benchmarking
			_, _ = stubRuntime.ExtractEntities(context.Background(), telemetryItem)
		}
	})
}

// BenchmarkWasmVsStub_FullPipeline compares WASM vs stub for the full processing pipeline
func BenchmarkWasmVsStub_FullPipeline(b *testing.B) {
	// Skip if not fullwasm build
	if os.Getenv("FULLWASM_TEST") != "1" {
		b.Skip("Skipping WASM benchmark - requires FULLWASM_TEST=1 environment variable")
	}

	// Find WASM models
	repoRoot := filepath.Join("..", "..", "..")
	errorClassifierPath := filepath.Join(repoRoot, "wasm-models", "error-classifier", "build", "error-classifier.wasm")
	samplerPath := filepath.Join(repoRoot, "wasm-models", "importance-sampler", "build", "importance-sampler.wasm")
	entityExtractorPath := filepath.Join(repoRoot, "wasm-models", "entity-extractor", "build", "entity-extractor.wasm")

	// Verify WASM models exist
	for _, path := range []string{errorClassifierPath, samplerPath, entityExtractorPath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			b.Fatalf("WASM model not found at %s", path)
		}
	}

	// Create loggers
	silentLogger := zap.NewNop()
	
	// Create processor configurations
	fullWasmConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ErrorClassifier: processor.ModelConfig{
				Path:          errorClassifierPath,
				MemoryLimitMB: 100,
				TimeoutMs:     50,
			},
			ImportanceSampler: processor.ModelConfig{
				Path:          samplerPath,
				MemoryLimitMB: 80,
				TimeoutMs:     30,
			},
			EntityExtractor: processor.ModelConfig{
				Path:          entityExtractorPath,
				MemoryLimitMB: 150,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    true,
		},
		Processing: processor.ProcessingConfig{
			BatchSize:    10,
			Concurrency:  2,
			QueueSize:    100,
			TimeoutMs:    250,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}
	
	stubConfig := &processor.Config{
		Models: processor.ModelsConfig{
			ErrorClassifier: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 100,
				TimeoutMs:     50,
			},
			ImportanceSampler: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 80,
				TimeoutMs:     30,
			},
			EntityExtractor: processor.ModelConfig{
				Path:          "stub",
				MemoryLimitMB: 150,
				TimeoutMs:     50,
			},
		},
		Features: processor.FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    true,
		},
		Processing: processor.ProcessingConfig{
			BatchSize:    10,
			Concurrency:  2,
			QueueSize:    100,
			TimeoutMs:    250,
		},
		Output: processor.OutputConfig{
			AttributeNamespace:      "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}

	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger:            silentLogger,
		BuildInfo:         component.BuildInfo{},
		TelemetrySettings: component.TelemetrySettings{},
	}

	// Create mixed test data with 50 spans for a realistic workload
	traces := createManyTestTraces("mixed-services", 50)

	// Benchmark the WASM implementation
	b.Run("WASM", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			fullWasmConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create WASM processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			err := proc.ConsumeTraces(context.Background(), traces)
			if err != nil {
				b.Fatalf("Error processing traces: %v", err)
			}
		}
	})

	// Benchmark the stub implementation
	b.Run("Stub", func(b *testing.B) {
		// Create the factory
		factory := processor.NewFactory()

		// Create a processor
		proc, err := factory.CreateTracesProcessor(
			context.Background(),
			settings,
			stubConfig,
			&MockTracesConsumer{},
		)
		if err != nil {
			b.Fatalf("Failed to create stub processor: %v", err)
		}
		defer proc.Shutdown(context.Background())

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			err := proc.ConsumeTraces(context.Background(), traces)
			if err != nil {
				b.Fatalf("Error processing traces: %v", err)
			}
		}
	})
}

// Helper function to extract attributes from a map
func extractAttributes(am pcommon.Map) map[string]interface{} {
	result := make(map[string]interface{})
	am.Range(func(k string, v pcommon.Value) bool {
		switch v.Type() {
		case pcommon.ValueTypeStr:
			result[k] = v.Str()
		case pcommon.ValueTypeInt:
			result[k] = v.Int()
		case pcommon.ValueTypeDouble:
			result[k] = v.Double()
		case pcommon.ValueTypeBool:
			result[k] = v.Bool()
		}
		return true
	})
	return result
}