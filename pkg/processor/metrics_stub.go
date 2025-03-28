//go:build !fullwasm
// +build !fullwasm

// This file contains the stub implementation of the metrics processor

package processor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
)

type stubMetricsProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Metrics
	wasmRuntime  *runtime.WasmRuntime
}

func newMetricsProcessor(
	logger *zap.Logger,
	config *Config,
	nextConsumer consumer.Metrics,
) (metricsProcessor, error) {
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

	return &stubMetricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  wasmRuntime,
	}, nil
}

func (p *stubMetricsProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// Stub implementation just passes metrics through
	p.logger.Debug("Stub metrics processor called", 
		zap.Int("metric_count", md.MetricCount()))
	return md, nil
}

func (p *stubMetricsProcessor) shutdown(ctx context.Context) error {
	return p.wasmRuntime.Close()
}