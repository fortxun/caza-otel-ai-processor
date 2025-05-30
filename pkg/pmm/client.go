package pmm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
	cache      *MetricsCache
}

type Config struct {
	BaseURL     string        `mapstructure:"base_url"`
	APIKey      string        `mapstructure:"api_key"`
	Timeout     time.Duration `mapstructure:"timeout"`
	RetryCount  int           `mapstructure:"retry_count"`
	MaxRetryTime time.Duration `mapstructure:"max_retry_time"`
	CacheTTL    time.Duration `mapstructure:"cache_ttl"`
	CacheSize   int           `mapstructure:"cache_size"`
}

func NewClient(config *Config, logger *zap.Logger) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("PMM base URL cannot be empty")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cacheSize := config.CacheSize
	if cacheSize == 0 {
		cacheSize = 1000
	}

	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}

	return &Client{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
		cache:  NewMetricsCache(cacheSize, cacheTTL),
	}, nil
}

type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values,omitempty"`
			Value  []interface{}     `json:"value,omitempty"`
		} `json:"result"`
	} `json:"data"`
}

func (c *Client) QueryPromQL(ctx context.Context, query string, timestamp time.Time) (*QueryResult, error) {
	cacheKey := fmt.Sprintf("%s:%d", query, timestamp.Unix())
	if cachedResult := c.cache.Get(cacheKey); cachedResult != nil {
		c.logger.Debug("Using cached PromQL result", 
			zap.String("query", query),
			zap.Time("timestamp", timestamp))
		return cachedResult, nil
	}

	queryURL, err := url.Parse(fmt.Sprintf("%s/api/v1/query", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PMM URL: %w", err)
	}

	params := url.Values{}
	params.Add("query", query)
	if !timestamp.IsZero() {
		params.Add("time", strconv.FormatFloat(float64(timestamp.Unix())+float64(timestamp.Nanosecond())/1e9, 'f', -1, 64))
	}
	queryURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	var resp *http.Response
	retryBackoff := backoff.NewExponentialBackOff()
	retryBackoff.MaxElapsedTime = 30 * time.Second

	err = backoff.Retry(func() error {
		var reqErr error
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil {
			c.logger.Warn("PromQL query request failed, retrying", 
				zap.String("query", query),
				zap.Error(reqErr))
			return reqErr
		}
		
		if resp.StatusCode >= 500 {
			reqErr = fmt.Errorf("server error: %s", resp.Status)
			c.logger.Warn("PromQL query returned server error, retrying", 
				zap.String("query", query),
				zap.Int("status_code", resp.StatusCode))
			return reqErr
		}
		
		return nil
	}, retryBackoff)

	if err != nil {
		return nil, fmt.Errorf("failed to execute PromQL query after retries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PMM API returned non-200 status: %d %s: %s", 
			resp.StatusCode, resp.Status, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode PromQL response: %w", err)
	}

	c.cache.Set(cacheKey, &result)

	return &result, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	cacheKey := fmt.Sprintf("%s:%d:%d:%d", query, start.Unix(), end.Unix(), step.Milliseconds())
	if cachedResult := c.cache.Get(cacheKey); cachedResult != nil {
		c.logger.Debug("Using cached PromQL range result", 
			zap.String("query", query),
			zap.Time("start", start),
			zap.Time("end", end))
		return cachedResult, nil
	}

	queryURL, err := url.Parse(fmt.Sprintf("%s/api/v1/query_range", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PMM URL: %w", err)
	}

	params := url.Values{}
	params.Add("query", query)
	params.Add("start", strconv.FormatFloat(float64(start.Unix())+float64(start.Nanosecond())/1e9, 'f', -1, 64))
	params.Add("end", strconv.FormatFloat(float64(end.Unix())+float64(end.Nanosecond())/1e9, 'f', -1, 64))
	params.Add("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))
	queryURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	var resp *http.Response
	retryBackoff := backoff.NewExponentialBackOff()
	retryBackoff.MaxElapsedTime = 30 * time.Second

	err = backoff.Retry(func() error {
		var reqErr error
		resp, reqErr = c.httpClient.Do(req)
		if reqErr != nil {
			c.logger.Warn("PromQL range query request failed, retrying", 
				zap.String("query", query),
				zap.Error(reqErr))
			return reqErr
		}
		
		if resp.StatusCode >= 500 {
			reqErr = fmt.Errorf("server error: %s", resp.Status)
			c.logger.Warn("PromQL range query returned server error, retrying", 
				zap.String("query", query),
				zap.Int("status_code", resp.StatusCode))
			return reqErr
		}
		
		return nil
	}, retryBackoff)

	if err != nil {
		return nil, fmt.Errorf("failed to execute PromQL range query after retries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PMM API returned non-200 status: %d %s: %s", 
			resp.StatusCode, resp.Status, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode PromQL response: %w", err)
	}

	c.cache.Set(cacheKey, &result)

	return &result, nil
}

func (c *Client) Close() error {
	return nil
}
