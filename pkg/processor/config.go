package processor

// Config defines the configuration for the AI processor.
type Config struct {
	// In newer versions, we use component.Config instead of configmodels.ProcessorSettings
	TypeVal string `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
	
	// Models configuration for AI models
	Models ModelsConfig `mapstructure:"models"`
	
	// Processing settings for batching and concurrency
	Processing ProcessingConfig `mapstructure:"processing"`
	
	// Features toggle for enabling/disabling specific features
	Features FeaturesConfig `mapstructure:"features"`
	
	// Sampling configuration for smart sampling
	Sampling SamplingConfig `mapstructure:"sampling"`
	
	// Output configuration for how AI-generated data is presented
	Output OutputConfig `mapstructure:"output"`
}

// ModelsConfig defines the configuration for the AI models.
type ModelsConfig struct {
	ErrorClassifier   ModelConfig `mapstructure:"error_classifier"`
	ImportanceSampler ModelConfig `mapstructure:"importance_sampler"`
	EntityExtractor   ModelConfig `mapstructure:"entity_extractor"`
}

// ModelConfig defines the configuration for an individual AI model.
type ModelConfig struct {
	// Path to the WASM model file
	Path string `mapstructure:"path"`
	
	// Memory limit in MB for the WASM module
	MemoryLimitMB int `mapstructure:"memory_limit_mb"`
	
	// Timeout in milliseconds for model inference
	TimeoutMs int `mapstructure:"timeout_ms"`
}

// ProcessingConfig defines the processing settings.
type ProcessingConfig struct {
	// BatchSize defines how many telemetry items to process in a batch
	BatchSize int `mapstructure:"batch_size"`
	
	// Concurrency defines how many concurrent model executions to run
	Concurrency int `mapstructure:"concurrency"`
	
	// QueueSize defines the maximum queue size for pending telemetry
	QueueSize int `mapstructure:"queue_size"`
	
	// TimeoutMs defines the overall timeout for processing a batch
	TimeoutMs int `mapstructure:"timeout_ms"`
	
	// EnableParallelProcessing enables processing telemetry items in parallel
	EnableParallelProcessing bool `mapstructure:"enable_parallel_processing"`
	
	// MaxParallelWorkers defines the maximum number of workers for parallel processing
	MaxParallelWorkers int `mapstructure:"max_parallel_workers"`
	
	// AttributeCacheSize defines the size of the attribute cache (0 to disable)
	AttributeCacheSize int `mapstructure:"attribute_cache_size"`
	
	// ResourceCacheSize defines the size of the resource cache (0 to disable)
	ResourceCacheSize int `mapstructure:"resource_cache_size"`
	
	// ModelCacheResults controls whether to cache model results for similar inputs
	ModelCacheResults bool `mapstructure:"model_cache_results"`
	
	// ModelResultsCacheSize defines the size of the model results cache per model
	ModelResultsCacheSize int `mapstructure:"model_results_cache_size"`
}

// FeaturesConfig defines which features are enabled.
type FeaturesConfig struct {
	// ErrorClassification enables error classification
	ErrorClassification bool `mapstructure:"error_classification"`
	
	// SmartSampling enables intelligent sampling
	SmartSampling bool `mapstructure:"smart_sampling"`
	
	// EntityExtraction enables extraction of entities from telemetry
	EntityExtraction bool `mapstructure:"entity_extraction"`
	
	// ContextLinking enables linking related telemetry items
	ContextLinking bool `mapstructure:"context_linking"`
}

// SamplingConfig defines the sampling configuration.
type SamplingConfig struct {
	// ErrorEvents sampling rate (0.0-1.0)
	ErrorEvents float64 `mapstructure:"error_events"`
	
	// SlowSpans sampling rate (0.0-1.0)
	SlowSpans float64 `mapstructure:"slow_spans"`
	
	// NormalSpans sampling rate (0.0-1.0)
	NormalSpans float64 `mapstructure:"normal_spans"`
	
	// ThresholdMs defines the threshold in ms for slow spans
	ThresholdMs int `mapstructure:"threshold_ms"`
}

// OutputConfig defines how the AI-generated data is presented.
type OutputConfig struct {
	// AttributeNamespace defines the attribute namespace for AI-generated attributes
	AttributeNamespace string `mapstructure:"attribute_namespace"`
	
	// IncludeConfidenceScores indicates whether to include confidence scores
	IncludeConfidenceScores bool `mapstructure:"include_confidence_scores"`
	
	// MaxAttributeLength defines the maximum length for AI-generated attributes
	MaxAttributeLength int `mapstructure:"max_attribute_length"`
}