//go:build !fullwasm
// +build !fullwasm

// This file contains the stub implementation of the traces processor

package processor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
)

type stubTracesProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Traces
	wasmRuntime  *runtime.WasmRuntime
}

func newTracesProcessor(
	logger *zap.Logger,
	config *Config,
	nextConsumer consumer.Traces,
) (tracesProcessor, error) {
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
		return nil, err
	}

	return &stubTracesProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  wasmRuntime,
	}, nil
}

func (p *stubTracesProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// Stub implementation just passes traces through
	p.logger.Debug("Stub traces processor called", 
		zap.Int("span_count", td.SpanCount()))
	return td, nil
}

func (p *stubTracesProcessor) shutdown(ctx context.Context) error {
	return p.wasmRuntime.Close()
}