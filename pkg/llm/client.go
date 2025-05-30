package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/types/config"
	"go.uber.org/zap"
)

type Client struct {
	config     *config.LLMConfig
	logger     *zap.Logger
	httpClient *http.Client
	cache      *ResponseCache
}

type Response struct {
	Content string `json:"content"`
	Model string `json:"model"`
	TokensUsed int `json:"tokens_used"`
	Timestamp time.Time `json:"timestamp"`
}

func NewClient(config *config.LLMConfig, logger *zap.Logger) (*Client, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("LLM integration is not enabled")
	}

	if config.Provider == "" {
		return nil, fmt.Errorf("LLM provider cannot be empty")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("LLM API key cannot be empty")
	}

	cacheSize := config.CacheSize
	if cacheSize <= 0 {
		cacheSize = 1000
	}

	cacheTTL := 24 * time.Hour // Default to 24 hours
	if config.CacheTTL != "" {
		var err error
		cacheTTL, err = time.ParseDuration(config.CacheTTL)
		if err != nil {
			logger.Warn("Invalid cache TTL format, using default",
				zap.String("ttl", config.CacheTTL),
				zap.Error(err))
		}
	}

	return &Client{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: NewResponseCache(cacheSize, cacheTTL),
	}, nil
}

func (c *Client) GenerateResponse(ctx context.Context, prompt string) (*Response, error) {
	if c.config.CacheEnabled {
		if cachedResponse := c.cache.Get(prompt); cachedResponse != nil {
			c.logger.Debug("Using cached LLM response",
				zap.String("prompt_prefix", prompt[:min(20, len(prompt))]))
			return cachedResponse, nil
		}
	}

	var response *Response
	var err error

	switch strings.ToLower(c.config.Provider) {
	case "openai":
		response, err = c.generateOpenAIResponse(ctx, prompt)
	case "anthropic":
		response, err = c.generateAnthropicResponse(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", c.config.Provider)
	}

	if err != nil {
		return nil, err
	}

	if c.config.CacheEnabled {
		c.cache.Set(prompt, response)
	}

	return response, nil
}

func (c *Client) generateOpenAIResponse(ctx context.Context, prompt string) (*Response, error) {
	type OpenAIMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type OpenAIRequest struct {
		Model       string         `json:"model"`
		Messages    []OpenAIMessage `json:"messages"`
		MaxTokens   int            `json:"max_tokens,omitempty"`
		Temperature float32        `json:"temperature"`
	}

	model := c.config.Model
	if model == "" {
		model = "gpt-4"
	}

	maxTokens := c.config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	requestBody := OpenAIRequest{
		Model: model,
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: "You are an expert database administrator analyzing database metrics and anomalies. Provide concise, technical analysis and recommendations.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: c.config.Temperature,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute OpenAI request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned non-200 status: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	var openAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Model   string `json:"model"`
		Usage   struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openAIResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	if len(openAIResponse.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned empty response")
	}

	return &Response{
		Content:    openAIResponse.Choices[0].Message.Content,
		Model:      openAIResponse.Model,
		TokensUsed: openAIResponse.Usage.TotalTokens,
		Timestamp:  time.Now(),
	}, nil
}

func (c *Client) generateAnthropicResponse(ctx context.Context, prompt string) (*Response, error) {
	type AnthropicRequest struct {
		Prompt       string  `json:"prompt"`
		Model        string  `json:"model"`
		MaxTokens    int     `json:"max_tokens_to_sample"`
		Temperature  float32 `json:"temperature"`
		StopSequences []string `json:"stop_sequences"`
	}

	model := c.config.Model
	if model == "" {
		model = "claude-2"
	}

	maxTokens := c.config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	formattedPrompt := fmt.Sprintf("\n\nHuman: %s\n\nAssistant:", prompt)

	requestBody := AnthropicRequest{
		Prompt:       formattedPrompt,
		Model:        model,
		MaxTokens:    maxTokens,
		Temperature:  c.config.Temperature,
		StopSequences: []string{"\n\nHuman:"},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/complete", bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Anthropic request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned non-200 status: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	var anthropicResponse struct {
		Completion string `json:"completion"`
		StopReason string `json:"stop_reason"`
		Model      string `json:"model"`
	}

	if err := json.Unmarshal(body, &anthropicResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	return &Response{
		Content:    strings.TrimSpace(anthropicResponse.Completion),
		Model:      anthropicResponse.Model,
		TokensUsed: 0, // Anthropic doesn't provide token usage
		Timestamp:  time.Now(),
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
