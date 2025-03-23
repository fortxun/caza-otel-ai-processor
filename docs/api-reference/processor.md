# Processor API Reference

This document provides detailed information about the API interfaces of the AI-Enhanced Telemetry Processor.

## Overview

The AI-Enhanced Telemetry Processor implements the OpenTelemetry Processor interface, which consists of three main processing functions:

1. `ProcessTraces`: For processing trace data
2. `ProcessMetrics`: For processing metric data
3. `ProcessLogs`: For processing log data

Each of these functions takes a context and telemetry data as input and returns processed data and an error (if any).

## Core Interfaces

### Factory Interface

The processor factory creates and manages processor instances:

```go
// Factory creates AI processor instances.
type Factory struct {
	processors *sync.Map
}

// NewFactory creates a new Factory.
func NewFactory() *Factory

// CreateTracesProcessor creates a trace processor.
func (f *Factory) CreateTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error)

// CreateMetricsProcessor creates a metrics processor.
func (f *Factory) CreateMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error)

// CreateLogsProcessor creates a logs processor.
func (f *Factory) CreateLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error)
```

### Configuration Interface

The processor configuration defines all settings:

```go
// Config defines the configuration for the AI processor.
type Config struct {
	// Models configuration
	Models ModelsConfig `mapstructure:"models"`

	// Processing settings
	Processing ProcessingConfig `mapstructure:"processing"`

	// Feature toggles
	Features FeaturesConfig `mapstructure:"features"`

	// Sampling configuration
	Sampling SamplingConfig `mapstructure:"sampling"`

	// Output configuration
	Output OutputConfig `mapstructure:"output"`
}

// ModelsConfig contains configuration for WASM models.
type ModelsConfig struct {
	ErrorClassifier  ModelConfig `mapstructure:"error_classifier"`
	ImportanceSampler ModelConfig `mapstructure:"importance_sampler"`
	EntityExtractor  ModelConfig `mapstructure:"entity_extractor"`
}

// ModelConfig defines configuration for a single WASM model.
type ModelConfig struct {
	Path           string `mapstructure:"path"`
	MemoryLimitMB  int    `mapstructure:"memory_limit_mb"`
	TimeoutMS      int    `mapstructure:"timeout_ms"`
	CacheSize      int    `mapstructure:"cache_size"`
}

// ProcessingConfig defines processing parameters.
type ProcessingConfig struct {
	BatchSize    int `mapstructure:"batch_size"`
	Concurrency  int `mapstructure:"concurrency"`
	QueueSize    int `mapstructure:"queue_size"`
	TimeoutMS    int `mapstructure:"timeout_ms"`
}

// FeaturesConfig controls which features are enabled.
type FeaturesConfig struct {
	ErrorClassification bool `mapstructure:"error_classification"`
	SmartSampling       bool `mapstructure:"smart_sampling"`
	EntityExtraction    bool `mapstructure:"entity_extraction"`
	ContextLinking      bool `mapstructure:"context_linking"`
}

// SamplingConfig defines sampling behavior.
type SamplingConfig struct {
	ErrorEvents  float64 `mapstructure:"error_events"`
	SlowSpans    float64 `mapstructure:"slow_spans"`
	NormalSpans  float64 `mapstructure:"normal_spans"`
	ThresholdMS  int     `mapstructure:"threshold_ms"`
}

// OutputConfig controls how processed data is output.
type OutputConfig struct {
	AttributeNamespace     string `mapstructure:"attribute_namespace"`
	IncludeConfidenceScores bool   `mapstructure:"include_confidence_scores"`
	MaxAttributeLength      int    `mapstructure:"max_attribute_length"`
}
```

### Processor Interface

The main processor implementation:

```go
// AIProcessor implements the OpenTelemetry processor interface.
type AIProcessor struct {
	config       *Config
	nextConsumer consumer.Traces | consumer.Metrics | consumer.Logs
	wasmRuntime  runtime.WASMRuntime
	workerPool   *ParallelProcessor
	logger       *zap.Logger
	cache        *runtime.ResultCache
}

// ProcessTraces processes trace data.
func (p *AIProcessor) ProcessTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error)

// ProcessMetrics processes metric data.
func (p *AIProcessor) ProcessMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error)

// ProcessLogs processes log data.
func (p *AIProcessor) ProcessLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error)

// Start starts the processor.
func (p *AIProcessor) Start(ctx context.Context) error

// Shutdown stops the processor.
func (p *AIProcessor) Shutdown(ctx context.Context) error
```

## WASM Runtime Interface

The WASM runtime interface abstracts the interaction with WASM models:

```go
// WASMRuntime provides an interface to interact with WASM models.
type WASMRuntime interface {
	// LoadModel loads a WASM model from a file.
	LoadModel(name string, path string, memoryLimitMB int) error

	// UnloadModel unloads a WASM model.
	UnloadModel(name string) error

	// CallFunction calls a function in a WASM model.
	CallFunction(modelName string, functionName string, input string, timeoutMS int) (string, error)

	// Close releases resources.
	Close() error
}
```

## Parallel Processing Interface

The parallel processing interface manages concurrent execution:

```go
// ParallelProcessor handles parallel processing of telemetry items.
type ParallelProcessor struct {
	workers     int
	queue       chan ProcessTask
	waitGroup   sync.WaitGroup
	stopChannel chan struct{}
	logger      *zap.Logger
}

// ProcessTask represents a task to be processed in parallel.
type ProcessTask struct {
	Context   context.Context
	Input     interface{}
	Process   ProcessFunc
	Result    chan ProcessResult
}

// ProcessFunc defines a function that processes a telemetry item.
type ProcessFunc func(context.Context, interface{}) (interface{}, error)

// ProcessResult contains the result of processing a telemetry item.
type ProcessResult struct {
	Output interface{}
	Error  error
}

// NewParallelProcessor creates a new parallel processor.
func NewParallelProcessor(workers int, queueSize int, logger *zap.Logger) *ParallelProcessor

// Start starts the processor's worker pool.
func (p *ParallelProcessor) Start()

// Stop stops the processor's worker pool.
func (p *ParallelProcessor) Stop()

// Process submits a task for processing.
func (p *ParallelProcessor) Process(ctx context.Context, input interface{}, process ProcessFunc) (interface{}, error)

// ProcessBatch processes multiple items in parallel.
func (p *ParallelProcessor) ProcessBatch(ctx context.Context, inputs []interface{}, process ProcessFunc) ([]interface{}, []error)
```

## Caching Interface

The caching interface provides result caching to avoid redundant model invocations:

```go
// ResultCache caches model execution results.
type ResultCache struct {
	caches     map[string]Cache
	maxSize    int
	logger     *zap.Logger
	mutex      sync.RWMutex
}

// Cache is a simple key-value cache with LRU eviction.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Len() int
}

// NewResultCache creates a new result cache.
func NewResultCache(models []string, maxSize int, logger *zap.Logger) *ResultCache

// Get retrieves a cached result or indicates a cache miss.
func (c *ResultCache) Get(model string, input string) (string, bool)

// Set stores a result in the cache.
func (c *ResultCache) Set(model string, input string, result string)
```

## Model Input/Output Interfaces

Each model has specific input and output formats:

### Error Classifier

```go
// ErrorClassifierInput represents input to the error classifier model.
type ErrorClassifierInput struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Kind       string                 `json:"kind"`
	Attributes map[string]interface{} `json:"attributes"`
	Resource   map[string]interface{} `json:"resource"`
}

// ErrorClassifierOutput represents output from the error classifier model.
type ErrorClassifierOutput struct {
	Category   string  `json:"category"`
	System     string  `json:"system"`
	Owner      string  `json:"owner"`
	Severity   string  `json:"severity"`
	Impact     string  `json:"impact"`
	Confidence float64 `json:"confidence"`
}
```

### Importance Sampler

```go
// ImportanceSamplerInput represents input to the importance sampler model.
type ImportanceSamplerInput struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Kind       string                 `json:"kind"`
	Duration   int64                  `json:"duration"`
	Attributes map[string]interface{} `json:"attributes"`
	Resource   map[string]interface{} `json:"resource"`
}

// ImportanceSamplerOutput represents output from the importance sampler model.
type ImportanceSamplerOutput struct {
	Importance float64 `json:"importance"`
	Keep       bool    `json:"keep"`
	Reason     string  `json:"reason"`
}
```

### Entity Extractor

```go
// EntityExtractorInput represents input to the entity extractor model.
type EntityExtractorInput struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Body        string                 `json:"body"`
	Attributes  map[string]interface{} `json:"attributes"`
	Resource    map[string]interface{} `json:"resource"`
}

// EntityExtractorOutput represents output from the entity extractor model.
type EntityExtractorOutput struct {
	Services    []string `json:"services"`
	Dependencies []string `json:"dependencies"`
	Operations  []string `json:"operations"`
	Confidence  float64  `json:"confidence"`
}
```

## Extension Points

The processor provides several extension points:

### Custom Model Integration

You can extend the processor by implementing additional WASM models:

1. Create a new model definition in the configuration:
```go
type ModelsConfig struct {
	// Existing models...
	CustomModel ModelConfig `mapstructure:"custom_model"`
}
```

2. Implement the model loading logic:
```go
func (p *AIProcessor) loadModels(ctx context.Context) error {
	// Existing model loading...
	if err := p.wasmRuntime.LoadModel("custom_model", p.config.Models.CustomModel.Path, p.config.Models.CustomModel.MemoryLimitMB); err != nil {
		return fmt.Errorf("failed to load custom model: %w", err)
	}
	return nil
}
```

3. Call the model in your processing logic:
```go
func (p *AIProcessor) processWithCustomModel(ctx context.Context, input string) (string, error) {
	return p.wasmRuntime.CallFunction("custom_model", "process_data", input, p.config.Models.CustomModel.TimeoutMS)
}
```

### Custom Processing Logic

You can extend the processor with custom processing logic:

1. Implement a new processing function:
```go
func (p *AIProcessor) customProcessing(ctx context.Context, span ptrace.Span) error {
	// Custom processing logic
	return nil
}
```

2. Hook it into the main processing pipeline:
```go
func (p *AIProcessor) processSpan(ctx context.Context, span ptrace.Span) error {
	// Existing processing...
	if err := p.customProcessing(ctx, span); err != nil {
		p.logger.Error("Custom processing failed", zap.Error(err))
	}
	return nil
}
```

### Custom Attribute Handling

You can extend the processor with custom attribute handling:

1. Implement a custom attribute processor:
```go
func (p *AIProcessor) processCustomAttributes(attributes pcommon.Map) {
	// Custom attribute processing
}
```

2. Hook it into the attribute processing pipeline:
```go
func (p *AIProcessor) processAttributes(attributes pcommon.Map) {
	// Existing attribute processing...
	p.processCustomAttributes(attributes)
}
```

## Error Handling

The processor provides error handling mechanisms:

```go
// HandleError decides how to handle an error.
func (p *AIProcessor) HandleError(ctx context.Context, err error) error {
	// Log the error
	p.logger.Error("Error during processing", zap.Error(err))
	
	// Decide whether to return the error or absorb it
	if isRetryableError(err) {
		return err
	}
	
	// Don't propagate non-retryable errors to avoid pipeline disruption
	return nil
}

// isRetryableError determines if an error should be retried.
func isRetryableError(err error) bool {
	// Error classification logic
	return false
}
```

## Metrics and Telemetry

The processor exports metrics for monitoring:

```go
var (
	// ProcessedItemsCounter counts processed telemetry items.
	ProcessedItemsCounter = metric.NewInt64Counter(
		"ai_processor_processed_items",
		metric.WithDescription("Number of telemetry items processed"),
		metric.WithUnit("1"),
	)

	// ProcessingDuration measures processing time.
	ProcessingDuration = metric.NewFloat64Histogram(
		"ai_processor_processing_duration",
		metric.WithDescription("Time spent processing telemetry items"),
		metric.WithUnit("ms"),
	)

	// ModelInvocationCounter counts model invocations.
	ModelInvocationCounter = metric.NewInt64Counter(
		"ai_processor_model_invocations",
		metric.WithDescription("Number of WASM model invocations"),
		metric.WithUnit("1"),
	)

	// ModelInvocationDuration measures model execution time.
	ModelInvocationDuration = metric.NewFloat64Histogram(
		"ai_processor_model_duration",
		metric.WithDescription("Time spent executing WASM models"),
		metric.WithUnit("ms"),
	)

	// CacheHitCounter counts cache hits.
	CacheHitCounter = metric.NewInt64Counter(
		"ai_processor_cache_hits",
		metric.WithDescription("Number of cache hits"),
		metric.WithUnit("1"),
	)

	// CacheMissCounter counts cache misses.
	CacheMissCounter = metric.NewInt64Counter(
		"ai_processor_cache_misses",
		metric.WithDescription("Number of cache misses"),
		metric.WithUnit("1"),
	)
)
```