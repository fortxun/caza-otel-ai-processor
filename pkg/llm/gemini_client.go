package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type GeminiClient struct {
	config     *config.LLMConfig
	logger     *zap.Logger
	httpClient *http.Client
	apiURL     string
	promptTmpl PromptTemplates
}

func NewGeminiClient(cfg *config.LLMConfig, logger *zap.Logger) (*GeminiClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &GeminiClient{
		config:     cfg,
		logger:     logger,
		httpClient: httpClient,
		apiURL:     "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent",
		promptTmpl: DefaultPromptTemplates(),
	}, nil
}

func (c *GeminiClient) GenerateRCA(ctx context.Context, anomalies []models.Anomaly, contextData ContextData) ([]models.RootCause, error) {
	prompt := c.buildRCAPrompt(anomalies, contextData)
	
	response, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	
	rootCauses, err := c.parseRCAResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RCA response: %w", err)
	}
	
	return rootCauses, nil
}

func (c *GeminiClient) GenerateSummary(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) (string, error) {
	prompt := c.buildSummaryPrompt(anomalies, rootCauses)
	
	response, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	
	summary := c.extractTextFromResponse(response)
	
	return summary, nil
}

func (c *GeminiClient) GenerateRecommendations(ctx context.Context, anomalies []models.Anomaly, rootCauses []models.RootCause) ([]string, error) {
	prompt := c.buildRecommendationsPrompt(anomalies, rootCauses)
	
	response, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	
	recommendations, err := c.parseRecommendationsResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recommendations response: %w", err)
	}
	
	return recommendations, nil
}

func (c *GeminiClient) buildRCAPrompt(anomalies []models.Anomaly, contextData ContextData) string {
	var sb strings.Builder
	
	prompt := c.promptTmpl.RCAPrompt
	
	prompt = strings.Replace(prompt, "{{ANOMALIES}}", c.formatAnomaliesForPrompt(anomalies), 1)
	prompt = strings.Replace(prompt, "{{RELATED_METRICS}}", c.formatRelatedMetricsForPrompt(contextData.RelatedMetrics), 1)
	prompt = strings.Replace(prompt, "{{SYSTEM_INFO}}", c.formatSystemInfoForPrompt(contextData.SystemInfo), 1)
	
	sb.WriteString(prompt)
	
	return sb.String()
}

func (c *GeminiClient) buildSummaryPrompt(anomalies []models.Anomaly, rootCauses []models.RootCause) string {
	var sb strings.Builder
	
	prompt := c.promptTmpl.SummaryPrompt
	
	prompt = strings.Replace(prompt, "{{ANOMALIES}}", c.formatAnomaliesForPrompt(anomalies), 1)
	prompt = strings.Replace(prompt, "{{ROOT_CAUSES}}", c.formatRootCausesForPrompt(rootCauses), 1)
	
	sb.WriteString(prompt)
	
	return sb.String()
}

func (c *GeminiClient) buildRecommendationsPrompt(anomalies []models.Anomaly, rootCauses []models.RootCause) string {
	var sb strings.Builder
	
	prompt := c.promptTmpl.RecommendationsPrompt
	
	prompt = strings.Replace(prompt, "{{ANOMALIES}}", c.formatAnomaliesForPrompt(anomalies), 1)
	prompt = strings.Replace(prompt, "{{ROOT_CAUSES}}", c.formatRootCausesForPrompt(rootCauses), 1)
	
	sb.WriteString(prompt)
	
	return sb.String()
}

func (c *GeminiClient) formatAnomaliesForPrompt(anomalies []models.Anomaly) string {
	var sb strings.Builder
	
	for i, anomaly := range anomalies {
		sb.WriteString(fmt.Sprintf("%d. Metric: %s (Type: %s)\n", i+1, anomaly.MetricName, anomaly.MetricType))
		sb.WriteString(fmt.Sprintf("   Value: %.2f %s (%.2f standard deviations from baseline of %.2f)\n", 
			anomaly.Value, anomaly.Unit, anomaly.DeviationScore, anomaly.Baseline))
		sb.WriteString(fmt.Sprintf("   Severity: %s, Timestamp: %s\n", 
			anomaly.Severity, anomaly.Timestamp.Format(time.RFC3339)))
		
		if len(anomaly.Labels) > 0 {
			sb.WriteString("   Labels: ")
			for k, v := range anomaly.Labels {
				sb.WriteString(fmt.Sprintf("%s=%s, ", k, v))
			}
			sb.WriteString("\n")
		}
		
		sb.WriteString("\n")
	}
	
	return sb.String()
}

func (c *GeminiClient) formatRelatedMetricsForPrompt(metrics []models.Metric) string {
	if len(metrics) == 0 {
		return "No related metrics available."
	}
	
	var sb strings.Builder
	
	for i, metric := range metrics {
		sb.WriteString(fmt.Sprintf("%d. %s: %.2f %s (Type: %s, Time: %s)\n", 
			i+1, metric.Name, metric.Value, metric.Unit, metric.Type, 
			metric.Timestamp.Format(time.RFC3339)))
	}
	
	return sb.String()
}

func (c *GeminiClient) formatSystemInfoForPrompt(info map[string]string) string {
	if len(info) == 0 {
		return "No system information available."
	}
	
	var sb strings.Builder
	
	for k, v := range info {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	
	return sb.String()
}

func (c *GeminiClient) formatRootCausesForPrompt(rootCauses []models.RootCause) string {
	if len(rootCauses) == 0 {
		return "No root causes identified yet."
	}
	
	var sb strings.Builder
	
	for i, rc := range rootCauses {
		sb.WriteString(fmt.Sprintf("%d. %s (Confidence: %.2f)\n", i+1, rc.Description, rc.Confidence))
		
		if len(rc.RelatedMetrics) > 0 {
			sb.WriteString("   Related Metrics: ")
			for j, metric := range rc.RelatedMetrics {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(metric)
			}
			sb.WriteString("\n")
		}
		
		if len(rc.Recommendations) > 0 {
			sb.WriteString("   Recommendations:\n")
			for _, rec := range rc.Recommendations {
				sb.WriteString(fmt.Sprintf("   - %s\n", rec))
			}
		}
		
		sb.WriteString("\n")
	}
	
	return sb.String()
}

func (c *GeminiClient) callGeminiAPI(ctx context.Context, prompt string) (string, error) {
	
	c.logger.Debug("Would call Gemini API with prompt",
		zap.String("prompt", prompt))
	
	
	return "This is a mock response from the Gemini API.", nil
}

func (c *GeminiClient) parseRCAResponse(response string) ([]models.RootCause, error) {
	
	mockRootCause := models.RootCause{
		Description: "High database load due to inefficient queries",
		Confidence: 0.85,
		RelatedMetrics: []string{"mysql_queries_per_second", "mysql_slow_queries"},
		Recommendations: []string{
			"Optimize the most frequent queries by adding appropriate indexes",
			"Consider implementing query caching for frequently accessed data",
		},
		LLMGenerated: true,
		Reasoning: "The correlation between high query rates and slow queries suggests inefficient query patterns.",
	}
	
	return []models.RootCause{mockRootCause}, nil
}

func (c *GeminiClient) parseRecommendationsResponse(response string) ([]string, error) {
	
	mockRecommendations := []string{
		"Optimize database queries by adding appropriate indexes",
		"Increase connection pool size to handle peak traffic",
		"Implement caching for frequently accessed data",
		"Consider scaling up database resources if load continues to increase",
	}
	
	return mockRecommendations, nil
}

func (c *GeminiClient) extractTextFromResponse(response string) string {
	return response
}
