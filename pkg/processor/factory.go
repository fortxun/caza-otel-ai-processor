// This file contains the common implementation of the processor factory
// that works with both stub and fullwasm builds

package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

const (
	// The value of "type" key in configuration.
	typeStr = "ai_processor"
)

// NewFactory creates a factory for the AI processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		processor.WithTraces(createTracesWrapper, component.StabilityLevelStable),
		processor.WithMetrics(createMetricsWrapper, component.StabilityLevelStable),
		processor.WithLogs(createLogsWrapper, component.StabilityLevelStable),
	)
}

// Create wrappers with the exact parameter types required by processor.CreateTracesFunc, etc.
func createTracesWrapper(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	pCfg := cfg.(*Config)
	
	// Create a new processor instance
	proc, err := newTracesProcessor(set.Logger, pCfg, nextConsumer)
	if err != nil {
		return nil, err
	}
	
	return &tracesProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}, nil
}

func createMetricsWrapper(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	
	// Create a new processor instance
	proc, err := newMetricsProcessor(set.Logger, pCfg, nextConsumer)
	if err != nil {
		return nil, err
	}
	
	return &metricsProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}, nil
}

func createLogsWrapper(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	pCfg := cfg.(*Config)
	
	// Create a new processor instance
	proc, err := newLogsProcessor(set.Logger, pCfg, nextConsumer)
	if err != nil {
		return nil, err
	}
	
	return &logsProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{
		TypeVal: typeStr,
		NameVal: typeStr,
		Models: ModelsConfig{
			ErrorClassifier: ModelConfig{
				Path:         "/models/error-classifier.wasm",
				MemoryLimitMB:  100,
				TimeoutMs:    50,
			},
			ImportanceSampler: ModelConfig{
				Path:         "/models/importance-sampler.wasm",
				MemoryLimitMB:  80,
				TimeoutMs:    30,
			},
			EntityExtractor: ModelConfig{
				Path:         "/models/entity-extractor.wasm",
				MemoryLimitMB:  150,
				TimeoutMs:    50,
			},
		},
		Processing: ProcessingConfig{
			BatchSize:             50,
			Concurrency:           4,
			QueueSize:             1000,
			TimeoutMs:             500,
			EnableParallelProcessing: true,
			MaxParallelWorkers:    8,
			AttributeCacheSize:    1000,
			ResourceCacheSize:     100,
			ModelCacheResults:     true,
			ModelResultsCacheSize: 1000,
		},
		Features: FeaturesConfig{
			ErrorClassification: true,
			SmartSampling:       true,
			EntityExtraction:    false,
			ContextLinking:      false,
		},
		Sampling: SamplingConfig{
			ErrorEvents:  1.0,
			SlowSpans:    1.0,
			NormalSpans:  0.1,
			ThresholdMs:  500,
		},
		Output: OutputConfig{
			AttributeNamespace:     "ai.",
			IncludeConfidenceScores: true,
			MaxAttributeLength:      256,
		},
	}
}

// tracesProcessorWrapper implements processor.Traces
type tracesProcessorWrapper struct {
	processor *tracesProcessor
	next      consumer.Traces
}

func (pw *tracesProcessorWrapper) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	processed, err := pw.processor.processTraces(ctx, td)
	if err != nil {
		return err
	}
	return pw.next.ConsumeTraces(ctx, processed)
}

func (pw *tracesProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *tracesProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *tracesProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}

// metricsProcessorWrapper implements processor.Metrics
type metricsProcessorWrapper struct {
	processor *metricsProcessor
	next      consumer.Metrics
}

func (pw *metricsProcessorWrapper) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	processed, err := pw.processor.processMetrics(ctx, md)
	if err != nil {
		return err
	}
	return pw.next.ConsumeMetrics(ctx, processed)
}

func (pw *metricsProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *metricsProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *metricsProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}

// logsProcessorWrapper implements processor.Logs
type logsProcessorWrapper struct {
	processor *logsProcessor
	next      consumer.Logs
}

func (pw *logsProcessorWrapper) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	processed, err := pw.processor.processLogs(ctx, ld)
	if err != nil {
		return err
	}
	return pw.next.ConsumeLogs(ctx, processed)
}

func (pw *logsProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *logsProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *logsProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}