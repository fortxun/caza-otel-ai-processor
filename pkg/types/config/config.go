package config

type Config struct {
	PMM      PMMConfig      `yaml:"pmm"`
	Analysis AnalysisConfig `yaml:"analysis"`
	Alerting AlertingConfig `yaml:"alerting"`
	Server   ServerConfig   `yaml:"server"`
}

type PMMConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  string `yaml:"timeout"`
}

type AnalysisConfig struct {
	Enabled           bool    `yaml:"enabled"`
	AnomalyThreshold  float64 `yaml:"anomaly_threshold"`
	SensitivityLevel  string  `yaml:"sensitivity_level"`
	MinDataPoints     int     `yaml:"min_data_points"`
	AdaptiveBaseline  bool    `yaml:"adaptive_baseline"`
	AdaptiveRate      float64 `yaml:"adaptive_rate"`
}

type AlertingConfig struct {
	Enabled bool         `yaml:"enabled"`
	Slack   SlackConfig  `yaml:"slack"`
	Email   EmailConfig  `yaml:"email"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
}

type EmailConfig struct {
	SMTPHost     string   `yaml:"smtp_host"`
	SMTPPort     int      `yaml:"smtp_port"`
	From         string   `yaml:"from"`
	To           []string `yaml:"to"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	UseTLS       bool     `yaml:"use_tls"`
}

type ServerConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	MetricsInterval string `yaml:"metrics_interval"`
}

type LLMConfig struct {
	Enabled      bool    `yaml:"enabled"`
	Provider     string  `yaml:"provider"`
	Model        string  `yaml:"model"`
	APIKey       string  `yaml:"api_key"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float32 `yaml:"temperature"`
	CacheEnabled bool    `yaml:"cache_enabled"`
	CacheSize    int     `yaml:"cache_size"`
	CacheTTL     string  `yaml:"cache_ttl"`
}
