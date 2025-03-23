//go:build !fullwasm
// +build !fullwasm

// This file contains the stub implementation of the logs processor

package processor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
)

type logsProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Logs
	wasmRuntime  *runtime.WasmRuntime
}

func newLogsProcessor(
	logger *zap.Logger,
	config *Config,
	nextConsumer consumer.Logs,
) (*logsProcessor, error) {
	// Initialize WASM runtime
	wasmRuntime, err := runtime.NewWasmRuntime(logger, &runtime.WasmRuntimeConfig{
		ErrorClassifierPath:   config.Models.ErrorClassifier.Path,
		ErrorClassifierMemory: config.Models.ErrorClassifier.MemoryLimit,
		SamplerPath:           config.Models.ImportanceSampler.Path,
		SamplerMemory:         config.Models.ImportanceSampler.MemoryLimit,
		EntityExtractorPath:   config.Models.EntityExtractor.Path,
		EntityExtractorMemory: config.Models.EntityExtractor.MemoryLimit,
	})
	if err != nil {
		return nil, err
	}

	return &logsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  wasmRuntime,
	}, nil
}

func (p *logsProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	// Stub implementation just passes logs through
	p.logger.Debug("Stub logs processor called", 
		zap.Int("log_record_count", ld.LogRecordCount()))
	return ld, nil
}

func (p *logsProcessor) shutdown(ctx context.Context) error {
	return p.wasmRuntime.Close()
}