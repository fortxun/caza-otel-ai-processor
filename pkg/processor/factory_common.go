// This file contains the common implementation of the processor factory
// that works with both stub and fullwasm builds

package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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
	
	wrapper := &tracesProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}
	return wrapper, nil
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
	
	wrapper := &metricsProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}
	return wrapper, nil
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
	
	wrapper := &logsProcessorWrapper{
		processor: proc,
		next:      nextConsumer,
	}
	return wrapper, nil
}

// CreateDefaultConfig creates the default configuration for the processor.
// Exported for testing purposes.
func CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

// createDefaultConfig creates the default configuration for the processor.
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