// This file contains the implementation of the logs processor with WASM support

package processor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
		ErrorClassifierMemory: config.Models.ErrorClassifier.MemoryLimitMB,
		SamplerPath:           config.Models.ImportanceSampler.Path,
		SamplerMemory:         config.Models.ImportanceSampler.MemoryLimitMB,
		EntityExtractorPath:   config.Models.EntityExtractor.Path,
		EntityExtractorMemory: config.Models.EntityExtractor.MemoryLimitMB,
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
	// If no AI features are enabled, pass through the data unchanged
	if !p.config.Features.ErrorClassification && 
	   !p.config.Features.SmartSampling && 
	   !p.config.Features.EntityExtraction {
		return ld, nil
	}

	// Use parallel processing if enabled
	if p.config.Processing.EnableParallelProcessing {
		return p.processLogsParallel(ctx, ld)
	}

	// Serial processing
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		sls := rl.ScopeLogs()
		
		for j := 0; j < sls.Len(); j++ {
			sl := sls.At(j)
			logs := sl.LogRecords()
			
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				p.processLogRecord(ctx, log, rl.Resource())
			}
		}
	}

	return ld, nil
}

// Process logs in parallel for better performance
func (p *logsProcessor) processLogsParallel(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	// Create a worker pool
	numWorkers := p.config.Processing.MaxParallelWorkers
	if numWorkers <= 0 {
		numWorkers = 8 // Default to 8 workers
	}
	pool := newWorkerPool(numWorkers)
	defer pool.close()

	// Process each resource log
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		sls := rl.ScopeLogs()
		
		for j := 0; j < sls.Len(); j++ {
			sl := sls.At(j)
			
			// Process logs in parallel
			processLogsInParallel(ctx, pool, sl.LogRecords(), rl.Resource(), p.processLogRecord)
		}
	}

	// Wait for all logs to be processed
	pool.wait()

	return ld, nil
}

func (p *logsProcessor) processLogRecord(ctx context.Context, log plog.LogRecord, resource pcommon.Resource) {
	// Extract information for classification
	logInfo := map[string]interface{}{
		"severity":    log.SeverityText(),
		"body":        log.Body().AsString(),
		"attributes":  attributesToMap(log.Attributes()),
		"resource":    attributesToMap(resource.Attributes()),
	}

	// Classify error logs if enabled
	if p.config.Features.ErrorClassification && log.SeverityNumber() >= plog.SeverityNumberError {
		p.classifyLogError(ctx, log, logInfo)
	}

	// Extract entities if enabled
	if p.config.Features.EntityExtraction {
		p.extractLogEntities(ctx, log, logInfo)
	}
}

func (p *logsProcessor) classifyLogError(ctx context.Context, log plog.LogRecord, logInfo map[string]interface{}) {
	// Call error classifier model
	result, err := p.wasmRuntime.ClassifyError(ctx, logInfo)
	if err != nil {
		p.logger.Error("Failed to classify log error", zap.Error(err))
		return
	}

	// Add classification attributes to log
	for k, v := range result {
		attrKey := p.config.Output.AttributeNamespace + k
		setAttribute(log.Attributes(), attrKey, v)
	}
}

func (p *logsProcessor) extractLogEntities(ctx context.Context, log plog.LogRecord, logInfo map[string]interface{}) {
	// Call entity extractor model
	result, err := p.wasmRuntime.ExtractEntities(ctx, logInfo)
	if err != nil {
		p.logger.Error("Failed to extract entities from log", zap.Error(err))
		return
	}

	// Add entity attributes to log
	for k, v := range result {
		attrKey := p.config.Output.AttributeNamespace + k
		setAttribute(log.Attributes(), attrKey, v)
	}
}

func (p *logsProcessor) shutdown(ctx context.Context) error {
	if p.wasmRuntime != nil {
		return p.wasmRuntime.Close()
	}
	return nil
}