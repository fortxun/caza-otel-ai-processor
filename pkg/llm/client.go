package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type Client interface {
	GenerateRCA(ctx context.Context, anomalies []models.Anomaly, contextData ContextData) ([]models.RootCause, error)
	GenerateSummary(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) (string, error)
	GenerateRecommendations(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) ([]string, error)
}

type ContextData struct {
	RelatedMetrics []models.Metric
	TimeRange      TimeRange
	SystemInfo     map[string]string
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type LLMManager struct {
	config     *config.LLMConfig
	logger     *zap.Logger
	client     Client
	cache      Cache
	promptTmpl PromptTemplates
}

func NewLLMManager(cfg *config.LLMConfig, logger *zap.Logger) (*LLMManager, error) {
	if cfg == nil {
		return nil, errors.New("LLM config cannot be nil")
	}

	if !cfg.Enabled {
		logger.Info("LLM integration is disabled")
		return nil, nil
	}

	var client Client
	var err error

	switch cfg.Provider {
	case "gemini":
		client, err = NewGeminiClient(cfg, logger)
	case "openai":
		client, err = NewOpenAIClient(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	var cache Cache
	if cfg.CacheEnabled {
		ttl, err := time.ParseDuration(cfg.CacheTTL)
		if err != nil {
			logger.Warn("Invalid cache TTL format, using default of 1 hour",
				zap.String("ttl", cfg.CacheTTL),
				zap.Error(err))
			ttl = 1 * time.Hour
		}
		cache = NewLRUCache(cfg.CacheSize, ttl)
	}

	return &LLMManager{
		config:     cfg,
		logger:     logger,
		client:     client,
		cache:      cache,
		promptTmpl: DefaultPromptTemplates(),
	}, nil
}

func (m *LLMManager) GenerateRCA(ctx context.Context, anomalies []models.Anomaly, contextData ContextData) ([]models.RootCause, error) {
	if !m.config.Enabled || m.client == nil {
		return nil, nil
	}

	cacheKey := m.generateCacheKey("rca", anomalies, contextData)

	if m.config.CacheEnabled && m.cache != nil {
		if cachedResult, found := m.cache.Get(cacheKey); found {
			m.logger.Debug("Using cached RCA result",
				zap.String("cache_key", cacheKey))
			return cachedResult.([]models.RootCause), nil
		}
	}

	rootCauses, err := m.client.GenerateRCA(ctx, anomalies, contextData)
	if err != nil {
		return nil, fmt.Errorf("LLM RCA generation failed: %w", err)
	}

	if m.config.CacheEnabled && m.cache != nil {
		ttl, _ := time.ParseDuration(m.config.CacheTTL)
		m.cache.Set(cacheKey, rootCauses, ttl)
	}

	return rootCauses, nil
}

func (m *LLMManager) GenerateSummary(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) (string, error) {
	if !m.config.Enabled || m.client == nil {
		return "", nil
	}

	cacheKey := m.generateCacheKey("summary", anomalies, ContextData{})

	if m.config.CacheEnabled && m.cache != nil {
		if cachedResult, found := m.cache.Get(cacheKey); found {
			m.logger.Debug("Using cached summary result",
				zap.String("cache_key", cacheKey))
			return cachedResult.(string), nil
		}
	}

	summary, err := m.client.GenerateSummary(ctx, anomalies, rootCauses)
	if err != nil {
		return "", fmt.Errorf("LLM summary generation failed: %w", err)
	}

	if m.config.CacheEnabled && m.cache != nil {
		ttl, _ := time.ParseDuration(m.config.CacheTTL)
		m.cache.Set(cacheKey, summary, ttl)
	}

	return summary, nil
}

func (m *LLMManager) GenerateRecommendations(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) ([]string, error) {
	if !m.config.Enabled || m.client == nil {
		return nil, nil
	}

	cacheKey := m.generateCacheKey("recommendations", anomalies, ContextData{})

	if m.config.CacheEnabled && m.cache != nil {
		if cachedResult, found := m.cache.Get(cacheKey); found {
			m.logger.Debug("Using cached recommendations result",
				zap.String("cache_key", cacheKey))
			return cachedResult.([]string), nil
		}
	}

	recommendations, err := m.client.GenerateRecommendations(ctx, anomalies, rootCauses)
	if err != nil {
		return nil, fmt.Errorf("LLM recommendations generation failed: %w", err)
	}

	if m.config.CacheEnabled && m.cache != nil {
		ttl, _ := time.ParseDuration(m.config.CacheTTL)
		m.cache.Set(cacheKey, recommendations, ttl)
	}

	return recommendations, nil
}

func (m *LLMManager) generateCacheKey(prefix string, anomalies []models.Anomaly, contextData ContextData) string {
	return generateHashKey(prefix, anomalies, contextData)
}
