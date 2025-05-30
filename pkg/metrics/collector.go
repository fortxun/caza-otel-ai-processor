package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/pmm"
	"go.uber.org/zap"
)

type MetricType string

const (
	MetricTypeLatency MetricType = "latency"
	MetricTypeTraffic MetricType = "traffic"
	MetricTypeError MetricType = "error"
	MetricTypeSaturation MetricType = "saturation"
)

type Metric struct {
	Name string `json:"name"`
	Type MetricType `json:"type"`
	Value float64 `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Labels map[string]string `json:"labels"`
	Unit string `json:"unit"`
}

type TimeSeriesMetric struct {
	Name string `json:"name"`
	Type MetricType `json:"type"`
	Values []TimeSeriesPoint `json:"values"`
	Labels map[string]string `json:"labels"`
	Unit string `json:"unit"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value float64 `json:"value"`
}

type Collector struct {
	pmmClient *pmm.Client
	logger    *zap.Logger
	config    *CollectorConfig
	metrics   map[string]Metric
	mutex     sync.RWMutex
	stopChan  chan struct{}
}

type CollectorConfig struct {
	MetricsQueries map[string]MetricQuery `json:"metrics_queries"`
	QueryInterval time.Duration `json:"query_interval"`
	DefaultLabels map[string]string `json:"default_labels"`
}

type MetricQuery struct {
	Query string `json:"query"`
	Type MetricType `json:"type"`
	Unit string `json:"unit"`
}

func NewCollector(pmmClient *pmm.Client, config *CollectorConfig, logger *zap.Logger) *Collector {
	return &Collector{
		pmmClient: pmmClient,
		logger:    logger,
		config:    config,
		metrics:   make(map[string]Metric),
		stopChan:  make(chan struct{}),
	}
}

func (c *Collector) Start(ctx context.Context) error {
	if c.config.QueryInterval == 0 {
		return fmt.Errorf("query interval must be greater than zero")
	}
	if len(c.config.MetricsQueries) == 0 {
		return fmt.Errorf("no metrics queries defined")
	}

	go c.collectLoop(ctx)
	return nil
}

func (c *Collector) Stop() {
	close(c.stopChan)
}

func (c *Collector) GetMetric(name string) (Metric, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metric, found := c.metrics[name]
	return metric, found
}

func (c *Collector) GetAllMetrics() []Metric {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metrics := make([]Metric, 0, len(c.metrics))
	for _, metric := range c.metrics {
		metrics = append(metrics, metric)
	}
	return metrics
}

func (c *Collector) GetMetricsByType(metricType MetricType) []Metric {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metrics := make([]Metric, 0)
	for _, metric := range c.metrics {
		if metric.Type == metricType {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

func (c *Collector) collectLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.QueryInterval)
	defer ticker.Stop()
	
	c.collectMetrics(ctx)
	
	for {
		select {
		case <-ticker.C:
			c.collectMetrics(ctx)
		case <-c.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Collector) collectMetrics(ctx context.Context) {
	c.logger.Debug("Collecting metrics from PMM")
	
	ctx, cancel := context.WithTimeout(ctx, c.config.QueryInterval/2)
	defer cancel()
	
	for name, metricQuery := range c.config.MetricsQueries {
		result, err := c.pmmClient.QueryPromQL(ctx, metricQuery.Query, time.Now())
		if err != nil {
			c.logger.Error("Failed to query metric",
				zap.String("metric", name),
				zap.Error(err))
			continue
		}
		
		if result.Status != "success" || len(result.Data.Result) == 0 {
			c.logger.Debug("No data returned for metric",
				zap.String("metric", name))
			continue
		}
		
		for _, series := range result.Data.Result {
			labels := make(map[string]string)
			
			for k, v := range c.config.DefaultLabels {
				labels[k] = v
			}
			
			for k, v := range series.Metric {
				labels[k] = v
			}
			
			var value float64
			var timestamp time.Time
			
			if len(series.Value) >= 2 {
				ts, ok := series.Value[0].(float64)
				if !ok {
					continue
				}
				timestamp = time.Unix(int64(ts), 0)
				
				val, ok := series.Value[1].(string)
				if !ok {
					continue
				}
				
				if _, err := fmt.Sscanf(val, "%f", &value); err != nil {
					continue
				}
			} else if len(series.Values) > 0 && len(series.Values[len(series.Values)-1]) >= 2 {
				lastPoint := series.Values[len(series.Values)-1]
				
				ts, ok := lastPoint[0].(float64)
				if !ok {
					continue
				}
				timestamp = time.Unix(int64(ts), 0)
				
				val, ok := lastPoint[1].(string)
				if !ok {
					continue
				}
				
				if _, err := fmt.Sscanf(val, "%f", &value); err != nil {
					continue
				}
			} else {
				c.logger.Debug("No value found in result",
					zap.String("metric", name))
				continue
			}
			
			metric := Metric{
				Name:      name,
				Type:      metricQuery.Type,
				Value:     value,
				Timestamp: timestamp,
				Labels:    labels,
				Unit:      metricQuery.Unit,
			}
			
			c.mutex.Lock()
			c.metrics[name] = metric
			c.mutex.Unlock()
			
			c.logger.Debug("Collected metric",
				zap.String("name", name),
				zap.Float64("value", value),
				zap.Time("timestamp", timestamp))
		}
	}
}

func DefaultDatabaseMetrics() map[string]MetricQuery {
	return map[string]MetricQuery{
		"mysql_query_latency_avg": {
			Query: `avg(mysql_global_status_query_time_total / mysql_global_status_queries) * 1000`,
			Type:  MetricTypeLatency,
			Unit:  "ms",
		},
		"mysql_query_latency_p95": {
			Query: `histogram_quantile(0.95, sum(rate(mysql_query_time_seconds_bucket[5m])) by (le))`,
			Type:  MetricTypeLatency,
			Unit:  "s",
		},
		
		"mysql_queries_per_second": {
			Query: `rate(mysql_global_status_queries[5m])`,
			Type:  MetricTypeTraffic,
			Unit:  "qps",
		},
		"mysql_connections": {
			Query: `mysql_global_status_threads_connected`,
			Type:  MetricTypeTraffic,
			Unit:  "connections",
		},
		
		"mysql_error_rate": {
			Query: `rate(mysql_global_status_queries_errors[5m]) / rate(mysql_global_status_queries[5m]) * 100`,
			Type:  MetricTypeError,
			Unit:  "percent",
		},
		"mysql_aborted_connections": {
			Query: `rate(mysql_global_status_aborted_connects[5m])`,
			Type:  MetricTypeError,
			Unit:  "connections/s",
		},
		
		"mysql_buffer_pool_usage": {
			Query: `mysql_global_status_innodb_buffer_pool_pages_data / mysql_global_status_innodb_buffer_pool_pages_total * 100`,
			Type:  MetricTypeSaturation,
			Unit:  "percent",
		},
		"mysql_cpu_usage": {
			Query: `sum by (instance) (rate(process_cpu_seconds_total{job="mysql"}[5m])) * 100`,
			Type:  MetricTypeSaturation,
			Unit:  "percent",
		},
		"mysql_memory_usage": {
			Query: `process_resident_memory_bytes{job="mysql"} / node_memory_MemTotal_bytes * 100`,
			Type:  MetricTypeSaturation,
			Unit:  "percent",
		},
		"mysql_disk_io_utilization": {
			Query: `rate(node_disk_io_time_seconds_total{device=~".*", job="node"}[5m]) * 100`,
			Type:  MetricTypeSaturation,
			Unit:  "percent",
		},
	}
}
