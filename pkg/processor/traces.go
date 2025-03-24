//go:build fullwasm
// +build fullwasm

// This file contains the full implementation of the traces processor with WASM support

package processor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
)

type tracesProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Traces
	wasmRuntime  *runtime.WasmRuntime
}

func newTracesProcessor(
	logger *zap.Logger,
	config *Config,
	nextConsumer consumer.Traces,
) (*tracesProcessor, error) {
	// Initialize WASM runtime
	wasmRuntime, err := runtime.NewWasmRuntime(logger, &runtime.WasmRuntimeConfig{
		ErrorClassifierPath:   config.Models.ErrorClassifier.Path,
		ErrorClassifierMemory: config.Models.ErrorClassifier.MemoryLimitMB,
		SamplerPath:           config.Models.ImportanceSampler.Path,
		SamplerMemory:         config.Models.ImportanceSampler.MemoryLimitMB,
		EntityExtractorPath:   config.Models.EntityExtractor.Path,
		EntityExtractorMemory: config.Models.EntityExtractor.MemoryLimitMB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize WASM runtime: %w", err)
	}

	return &tracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  wasmRuntime,
	}, nil
}

func (p *tracesProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// If no AI features are enabled, pass through the data unchanged
	if !p.config.Features.ErrorClassification && 
	   !p.config.Features.SmartSampling && 
	   !p.config.Features.EntityExtraction && 
	   !p.config.Features.ContextLinking {
		return td, nil
	}

	// Use parallel processing if enabled
	if p.config.Processing.EnableParallelProcessing {
		return p.processTracesParallel(ctx, td)
	}

	// Serial processing
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		sss := rs.ScopeSpans()
		
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				p.processSpan(ctx, span, rs.Resource())
			}
		}
	}

	// Apply sampling if enabled
	if p.config.Features.SmartSampling {
		td = p.sampleTraces(ctx, td)
	}

	return td, nil
}

// Process traces in parallel for better performance
func (p *tracesProcessor) processTracesParallel(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// Create a worker pool
	numWorkers := p.config.Processing.MaxParallelWorkers
	if numWorkers <= 0 {
		numWorkers = 8 // Default to 8 workers
	}
	pool := newWorkerPool(numWorkers)
	defer pool.close()

	// Process each resource span
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		sss := rs.ScopeSpans()
		
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			
			// Process spans in parallel
			processSpansInParallel(ctx, pool, ss.Spans(), rs.Resource(), p.processSpan)
		}
	}

	// Wait for all spans to be processed
	pool.wait()

	// Apply sampling if enabled
	if p.config.Features.SmartSampling {
		td = p.sampleTraces(ctx, td)
	}

	return td, nil
}

func (p *tracesProcessor) processSpan(ctx context.Context, span ptrace.Span, resource pcommon.Resource) {
	// Extract error information if this is an error span
	if span.Status().Code() == ptrace.StatusCodeError {
		if p.config.Features.ErrorClassification {
			p.classifyError(ctx, span, resource)
		}
	}

	// Extract entities if enabled
	if p.config.Features.EntityExtraction {
		p.extractEntities(ctx, span, resource)
	}
}

func (p *tracesProcessor) classifyError(ctx context.Context, span ptrace.Span, resource pcommon.Resource) {
	// Prepare error information for classification
	errorInfo := map[string]interface{}{
		"name":        span.Name(),
		"status":      span.Status().Message(),
		"kind":        span.Kind().String(),
		"attributes":  attributesToMap(span.Attributes()),
		"resource":    attributesToMap(resource.Attributes()),
	}

	// Call error classifier model
	result, err := p.wasmRuntime.ClassifyError(ctx, errorInfo)
	if err != nil {
		p.logger.Error("Failed to classify error", zap.Error(err))
		return
	}

	// Add classification attributes to span
	for k, v := range result {
		attrKey := p.config.Output.AttributeNamespace + k
		setAttribute(span.Attributes(), attrKey, v)
	}
}

func (p *tracesProcessor) extractEntities(ctx context.Context, span ptrace.Span, resource pcommon.Resource) {
	// Prepare span information for entity extraction
	spanInfo := map[string]interface{}{
		"name":        span.Name(),
		"attributes":  attributesToMap(span.Attributes()),
		"resource":    attributesToMap(resource.Attributes()),
	}

	// Call entity extractor model
	result, err := p.wasmRuntime.ExtractEntities(ctx, spanInfo)
	if err != nil {
		p.logger.Error("Failed to extract entities", zap.Error(err))
		return
	}

	// Add entity attributes to span
	for k, v := range result {
		attrKey := p.config.Output.AttributeNamespace + k
		setAttribute(span.Attributes(), attrKey, v)
	}
}

func (p *tracesProcessor) sampleTraces(ctx context.Context, td ptrace.Traces) ptrace.Traces {
	// Create a new Traces object to hold the sampled traces
	sampled := ptrace.NewTraces()
	
	// Process all resource spans
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		resource := rs.Resource()
		sss := rs.ScopeSpans()
		
		// Process spans for each scope
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			
			// Process each span
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				
				// Determine sampling decision
				keep := p.makeSamplingDecision(ctx, span, resource)
				
				if keep {
					// Add span to sampled traces
					newRS := getOrCreateResource(sampled, resource)
					newSS := getOrCreateScope(newRS, ss.Scope())
					newSpan := newSS.Spans().AppendEmpty()
					span.CopyTo(newSpan)
				}
			}
		}
	}
	
	return sampled
}

func (p *tracesProcessor) makeSamplingDecision(ctx context.Context, span ptrace.Span, resource pcommon.Resource) bool {
	// Always keep error spans if configured
	if span.Status().Code() == ptrace.StatusCodeError && p.config.Sampling.ErrorEvents >= 1.0 {
		return true
	}
	
	// Check if this is a slow span
	duration := span.EndTimestamp() - span.StartTimestamp()
	durationMs := int64(duration) / 1_000_000 // Convert nanoseconds to milliseconds
	
	if durationMs > int64(p.config.Sampling.ThresholdMs) && p.config.Sampling.SlowSpans >= 1.0 {
		return true
	}
	
	// Call the sampler model
	spanInfo := map[string]interface{}{
		"name":      span.Name(),
		"kind":      span.Kind().String(),
		"status":    span.Status().Code().String(),
		"duration":  durationMs,
		"attributes": attributesToMap(span.Attributes()),
		"resource":  attributesToMap(resource.Attributes()),
	}
	
	// Call importance sampler model
	result, err := p.wasmRuntime.SampleTelemetry(ctx, spanInfo)
	if err != nil {
		p.logger.Error("Failed to make sampling decision", zap.Error(err))
		// Default to the normal spans rate
		return randomSample(p.config.Sampling.NormalSpans)
	}
	
	importance, ok := result["importance"].(float64)
	if !ok {
		return randomSample(p.config.Sampling.NormalSpans)
	}
	
	// Make sampling decision based on importance
	// Higher importance means higher chance of keeping the span
	return randomSample(p.config.Sampling.NormalSpans * importance)
}

func (p *tracesProcessor) shutdown(ctx context.Context) error {
	if p.wasmRuntime != nil {
		return p.wasmRuntime.Close()
	}
	return nil
}

// Helper functions are now defined in the common package and imported via helpers.go