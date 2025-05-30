package pmm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       *auth
	logger     *zap.Logger
}

type auth struct {
	username string
	password string
}

func NewClient(config *config.PMMConfig, logger *zap.Logger) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("PMM configuration cannot be nil")
	}

	if config.URL == "" {
		return nil, fmt.Errorf("PMM URL cannot be empty")
	}

	timeout := 30 * time.Second
	if config.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			logger.Warn("Invalid timeout format, using default",
				zap.String("timeout", config.Timeout),
				zap.Error(err))
		}
	}

	return &Client{
		baseURL: strings.TrimSuffix(config.URL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		auth: &auth{
			username: config.Username,
			password: config.Password,
		},
		logger: logger,
	}, nil
}

func (c *Client) QueryMetrics(ctx context.Context, query string) ([]models.Metric, error) {
	c.logger.Debug("Querying PMM metrics", zap.String("query", query))

	apiURL := fmt.Sprintf("%s/api/v1/query", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	if c.auth.username != "" && c.auth.password != "" {
		req.SetBasicAuth(c.auth.username, c.auth.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PMM API returned non-200 status: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	var promResponse struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &promResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResponse.Status != "success" {
		return nil, fmt.Errorf("PMM API returned error status: %s", promResponse.Status)
	}

	metrics := make([]models.Metric, 0, len(promResponse.Data.Result))
	for _, result := range promResponse.Data.Result {
		metricName := result.Metric["__name__"]
		if metricName == "" {
			if job, ok := result.Metric["job"]; ok {
				metricName = job
			} else {
				metricName = "unknown"
			}
		}

		if len(result.Value) != 2 {
			c.logger.Warn("Unexpected value format in Prometheus response",
				zap.Any("value", result.Value))
			continue
		}

		timestampFloat, ok := result.Value[0].(float64)
		if !ok {
			c.logger.Warn("Failed to parse timestamp",
				zap.Any("timestamp", result.Value[0]))
			continue
		}
		timestamp := time.Unix(int64(timestampFloat), 0)

		valueStr, ok := result.Value[1].(string)
		if !ok {
			c.logger.Warn("Failed to parse value as string",
				zap.Any("value", result.Value[1]))
			continue
		}
		value, err := parseValue(valueStr)
		if err != nil {
			c.logger.Warn("Failed to parse value to float64",
				zap.String("value", valueStr),
				zap.Error(err))
			continue
		}

		labels := make(map[string]string)
		for k, v := range result.Metric {
			if k != "__name__" {
				labels[k] = v
			}
		}

		metricType := determineMetricType(metricName, labels)

		metric := models.Metric{
			Name:      metricName,
			Type:      metricType,
			Value:     value,
			Timestamp: timestamp,
			Labels:    labels,
			Unit:      determineUnit(metricName),
		}

		metrics = append(metrics, metric)
	}

	c.logger.Debug("Retrieved metrics from PMM",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]models.Metric, error) {
	c.logger.Debug("Querying PMM metrics range",
		zap.String("query", query),
		zap.Time("start", start),
		zap.Time("end", end),
		zap.Duration("step", step))

	apiURL := fmt.Sprintf("%s/api/v1/query_range", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", fmt.Sprintf("%d", start.Unix()))
	q.Add("end", fmt.Sprintf("%d", end.Unix()))
	q.Add("step", fmt.Sprintf("%ds", int(step.Seconds())))
	req.URL.RawQuery = q.Encode()

	if c.auth.username != "" && c.auth.password != "" {
		req.SetBasicAuth(c.auth.username, c.auth.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PMM API returned non-200 status: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	var promResponse struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &promResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResponse.Status != "success" {
		return nil, fmt.Errorf("PMM API returned error status: %s", promResponse.Status)
	}

	metrics := make([]models.Metric, 0)
	for _, result := range promResponse.Data.Result {
		metricName := result.Metric["__name__"]
		if metricName == "" {
			if job, ok := result.Metric["job"]; ok {
				metricName = job
			} else {
				metricName = "unknown"
			}
		}

		labels := make(map[string]string)
		for k, v := range result.Metric {
			if k != "__name__" {
				labels[k] = v
			}
		}

		metricType := determineMetricType(metricName, labels)
		unit := determineUnit(metricName)

		for _, valuePair := range result.Values {
			if len(valuePair) != 2 {
				c.logger.Warn("Unexpected value format in Prometheus response",
					zap.Any("value", valuePair))
				continue
			}

			timestampFloat, ok := valuePair[0].(float64)
			if !ok {
				c.logger.Warn("Failed to parse timestamp",
					zap.Any("timestamp", valuePair[0]))
				continue
			}
			timestamp := time.Unix(int64(timestampFloat), 0)

			valueStr, ok := valuePair[1].(string)
			if !ok {
				c.logger.Warn("Failed to parse value as string",
					zap.Any("value", valuePair[1]))
				continue
			}
			value, err := parseValue(valueStr)
			if err != nil {
				c.logger.Warn("Failed to parse value to float64",
					zap.String("value", valueStr),
					zap.Error(err))
				continue
			}

			metric := models.Metric{
				Name:      metricName,
				Type:      metricType,
				Value:     value,
				Timestamp: timestamp,
				Labels:    labels,
				Unit:      unit,
			}

			metrics = append(metrics, metric)
		}
	}

	c.logger.Debug("Retrieved metrics from PMM range query",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Client) GetGoldenSignalMetrics(ctx context.Context) ([]models.Metric, error) {
	latencyMetrics, err := c.GetLatencyMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latency metrics: %w", err)
	}

	trafficMetrics, err := c.GetTrafficMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic metrics: %w", err)
	}

	errorMetrics, err := c.GetErrorMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get error metrics: %w", err)
	}

	saturationMetrics, err := c.GetSaturationMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get saturation metrics: %w", err)
	}

	allMetrics := make([]models.Metric, 0)
	allMetrics = append(allMetrics, latencyMetrics...)
	allMetrics = append(allMetrics, trafficMetrics...)
	allMetrics = append(allMetrics, errorMetrics...)
	allMetrics = append(allMetrics, saturationMetrics...)

	return allMetrics, nil
}

func (c *Client) GetLatencyMetrics(ctx context.Context) ([]models.Metric, error) {
	mysqlLatencyQuery := `mysql_global_status_queries / rate(mysql_global_status_uptime[5m])`
	mysqlLatencyMetrics, err := c.QueryMetrics(ctx, mysqlLatencyQuery)
	if err != nil {
		c.logger.Warn("Failed to query MySQL latency metrics",
			zap.Error(err))
	}

	pgLatencyQuery := `pg_stat_activity_max_tx_duration`
	pgLatencyMetrics, err := c.QueryMetrics(ctx, pgLatencyQuery)
	if err != nil {
		c.logger.Warn("Failed to query PostgreSQL latency metrics",
			zap.Error(err))
	}

	mongoLatencyQuery := `mongodb_op_latency_latency`
	mongoLatencyMetrics, err := c.QueryMetrics(ctx, mongoLatencyQuery)
	if err != nil {
		c.logger.Warn("Failed to query MongoDB latency metrics",
			zap.Error(err))
	}

	latencyMetrics := make([]models.Metric, 0)
	for _, metric := range mysqlLatencyMetrics {
		metric.Type = models.MetricTypeLatency
		latencyMetrics = append(latencyMetrics, metric)
	}
	for _, metric := range pgLatencyMetrics {
		metric.Type = models.MetricTypeLatency
		latencyMetrics = append(latencyMetrics, metric)
	}
	for _, metric := range mongoLatencyMetrics {
		metric.Type = models.MetricTypeLatency
		latencyMetrics = append(latencyMetrics, metric)
	}

	return latencyMetrics, nil
}

func (c *Client) GetTrafficMetrics(ctx context.Context) ([]models.Metric, error) {
	mysqlConnectionsQuery := `mysql_global_status_threads_connected`
	mysqlConnectionsMetrics, err := c.QueryMetrics(ctx, mysqlConnectionsQuery)
	if err != nil {
		c.logger.Warn("Failed to query MySQL connections metrics",
			zap.Error(err))
	}

	mysqlQPSQuery := `rate(mysql_global_status_queries[5m])`
	mysqlQPSMetrics, err := c.QueryMetrics(ctx, mysqlQPSQuery)
	if err != nil {
		c.logger.Warn("Failed to query MySQL QPS metrics",
			zap.Error(err))
	}

	pgConnectionsQuery := `pg_stat_activity_count`
	pgConnectionsMetrics, err := c.QueryMetrics(ctx, pgConnectionsQuery)
	if err != nil {
		c.logger.Warn("Failed to query PostgreSQL connections metrics",
			zap.Error(err))
	}

	mongoConnectionsQuery := `mongodb_connections`
	mongoConnectionsMetrics, err := c.QueryMetrics(ctx, mongoConnectionsQuery)
	if err != nil {
		c.logger.Warn("Failed to query MongoDB connections metrics",
			zap.Error(err))
	}

	trafficMetrics := make([]models.Metric, 0)
	for _, metric := range mysqlConnectionsMetrics {
		metric.Type = models.MetricTypeTraffic
		trafficMetrics = append(trafficMetrics, metric)
	}
	for _, metric := range mysqlQPSMetrics {
		metric.Type = models.MetricTypeTraffic
		trafficMetrics = append(trafficMetrics, metric)
	}
	for _, metric := range pgConnectionsMetrics {
		metric.Type = models.MetricTypeTraffic
		trafficMetrics = append(trafficMetrics, metric)
	}
	for _, metric := range mongoConnectionsMetrics {
		metric.Type = models.MetricTypeTraffic
		trafficMetrics = append(trafficMetrics, metric)
	}

	return trafficMetrics, nil
}

func (c *Client) GetErrorMetrics(ctx context.Context) ([]models.Metric, error) {
	mysqlErrorsQuery := `mysql_global_status_connection_errors_total`
	mysqlErrorsMetrics, err := c.QueryMetrics(ctx, mysqlErrorsQuery)
	if err != nil {
		c.logger.Warn("Failed to query MySQL errors metrics",
			zap.Error(err))
	}

	pgDeadlocksQuery := `pg_stat_database_deadlocks`
	pgDeadlocksMetrics, err := c.QueryMetrics(ctx, pgDeadlocksQuery)
	if err != nil {
		c.logger.Warn("Failed to query PostgreSQL deadlocks metrics",
			zap.Error(err))
	}

	mongoAssertsQuery := `mongodb_asserts_total`
	mongoAssertsMetrics, err := c.QueryMetrics(ctx, mongoAssertsQuery)
	if err != nil {
		c.logger.Warn("Failed to query MongoDB asserts metrics",
			zap.Error(err))
	}

	errorMetrics := make([]models.Metric, 0)
	for _, metric := range mysqlErrorsMetrics {
		metric.Type = models.MetricTypeError
		errorMetrics = append(errorMetrics, metric)
	}
	for _, metric := range pgDeadlocksMetrics {
		metric.Type = models.MetricTypeError
		errorMetrics = append(errorMetrics, metric)
	}
	for _, metric := range mongoAssertsMetrics {
		metric.Type = models.MetricTypeError
		errorMetrics = append(errorMetrics, metric)
	}

	return errorMetrics, nil
}

func (c *Client) GetSaturationMetrics(ctx context.Context) ([]models.Metric, error) {
	cpuUsageQuery := `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	cpuUsageMetrics, err := c.QueryMetrics(ctx, cpuUsageQuery)
	if err != nil {
		c.logger.Warn("Failed to query CPU usage metrics",
			zap.Error(err))
	}

	memoryUsageQuery := `100 * (1 - ((node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes) / node_memory_MemTotal_bytes))`
	memoryUsageMetrics, err := c.QueryMetrics(ctx, memoryUsageQuery)
	if err != nil {
		c.logger.Warn("Failed to query memory usage metrics",
			zap.Error(err))
	}

	diskUsageQuery := `100 - ((node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100)`
	diskUsageMetrics, err := c.QueryMetrics(ctx, diskUsageQuery)
	if err != nil {
		c.logger.Warn("Failed to query disk usage metrics",
			zap.Error(err))
	}

	mysqlBufferPoolQuery := `mysql_global_status_innodb_buffer_pool_pages_total - mysql_global_status_innodb_buffer_pool_pages_free`
	mysqlBufferPoolMetrics, err := c.QueryMetrics(ctx, mysqlBufferPoolQuery)
	if err != nil {
		c.logger.Warn("Failed to query MySQL buffer pool metrics",
			zap.Error(err))
	}

	saturationMetrics := make([]models.Metric, 0)
	for _, metric := range cpuUsageMetrics {
		metric.Type = models.MetricTypeSaturation
		saturationMetrics = append(saturationMetrics, metric)
	}
	for _, metric := range memoryUsageMetrics {
		metric.Type = models.MetricTypeSaturation
		saturationMetrics = append(saturationMetrics, metric)
	}
	for _, metric := range diskUsageMetrics {
		metric.Type = models.MetricTypeSaturation
		saturationMetrics = append(saturationMetrics, metric)
	}
	for _, metric := range mysqlBufferPoolMetrics {
		metric.Type = models.MetricTypeSaturation
		saturationMetrics = append(saturationMetrics, metric)
	}

	return saturationMetrics, nil
}


func parseValue(value string) (float64, error) {
	var floatValue float64
	_, err := fmt.Sscanf(value, "%f", &floatValue)
	if err != nil {
		return 0, err
	}
	return floatValue, nil
}

func determineMetricType(name string, labels map[string]string) models.MetricType {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "latency") ||
		strings.Contains(nameLower, "duration") ||
		strings.Contains(nameLower, "response_time") ||
		strings.Contains(nameLower, "query_time") {
		return models.MetricTypeLatency
	}

	if strings.Contains(nameLower, "queries") ||
		strings.Contains(nameLower, "requests") ||
		strings.Contains(nameLower, "connections") ||
		strings.Contains(nameLower, "traffic") ||
		strings.Contains(nameLower, "throughput") {
		return models.MetricTypeTraffic
	}

	if strings.Contains(nameLower, "error") ||
		strings.Contains(nameLower, "exception") ||
		strings.Contains(nameLower, "failure") ||
		strings.Contains(nameLower, "deadlock") ||
		strings.Contains(nameLower, "timeout") {
		return models.MetricTypeError
	}

	if strings.Contains(nameLower, "usage") ||
		strings.Contains(nameLower, "utilization") ||
		strings.Contains(nameLower, "saturation") ||
		strings.Contains(nameLower, "capacity") ||
		strings.Contains(nameLower, "memory") ||
		strings.Contains(nameLower, "cpu") ||
		strings.Contains(nameLower, "disk") {
		return models.MetricTypeSaturation
	}

	return models.MetricTypeTraffic
}

func determineUnit(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "latency") ||
		strings.Contains(nameLower, "duration") ||
		strings.Contains(nameLower, "time") {
		return "seconds"
	}

	if strings.Contains(nameLower, "usage") ||
		strings.Contains(nameLower, "utilization") ||
		strings.Contains(nameLower, "percent") {
		return "%"
	}

	if strings.Contains(nameLower, "memory") ||
		strings.Contains(nameLower, "bytes") {
		return "bytes"
	}

	if strings.Contains(nameLower, "count") ||
		strings.Contains(nameLower, "total") ||
		strings.Contains(nameLower, "number") {
		return "count"
	}

	if strings.Contains(nameLower, "rate") ||
		strings.Contains(nameLower, "per_second") {
		return "per_second"
	}

	return ""
}
