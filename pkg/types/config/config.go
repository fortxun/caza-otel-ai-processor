package config

type LLMConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"`
	APIKey string `mapstructure:"api_key"`
	Model string `mapstructure:"model"`
	MaxTokens int `mapstructure:"max_tokens"`
	Temperature float32 `mapstructure:"temperature"`
	CacheEnabled bool `mapstructure:"cache_enabled"`
	CacheTTL string `mapstructure:"cache_ttl"`
	CacheSize int `mapstructure:"cache_size"`
	PromptTemplates map[string]string `mapstructure:"prompt_templates"`
}

type AnomalyDetectionConfig struct {
	Enabled bool `mapstructure:"enabled"`
	SensitivityLevel string `mapstructure:"sensitivity_level"`
	BaselineWindowHours int `mapstructure:"baseline_window_hours"`
	AnomalyThreshold float64 `mapstructure:"anomaly_threshold"`
	MinDataPoints int `mapstructure:"min_data_points"`
	SeasonalityPeriod int `mapstructure:"seasonality_period"`
	AdaptiveBaseline bool `mapstructure:"adaptive_baseline"`
	AdaptiveRate float64 `mapstructure:"adaptive_rate"`
}

type RootCauseAnalysisConfig struct {
	Enabled bool `mapstructure:"enabled"`
	CorrelationThreshold float64 `mapstructure:"correlation_threshold"`
	MaxCausalDepth int `mapstructure:"max_causal_depth"`
	KnowledgeBaseEnabled bool `mapstructure:"knowledge_base_enabled"`
	KnowledgeBasePath string `mapstructure:"knowledge_base_path"`
	UseLLM bool `mapstructure:"use_llm"`
	MaxRecommendations int `mapstructure:"max_recommendations"`
}

type PMMConfig struct {
	Enabled bool `mapstructure:"enabled"`
	BaseURL string `mapstructure:"base_url"`
	APIKey string `mapstructure:"api_key"`
	Timeout string `mapstructure:"timeout"`
	RetryCount int `mapstructure:"retry_count"`
	MaxRetryTime string `mapstructure:"max_retry_time"`
	CacheTTL string `mapstructure:"cache_ttl"`
	CacheSize int `mapstructure:"cache_size"`
	MetricsQueries map[string]string `mapstructure:"metrics_queries"`
	QueryInterval string `mapstructure:"query_interval"`
}
