package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/fortxun/idop/pkg/pmm"
	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
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
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
	Unit      string            `json:"unit"`
}

type Collector struct {
	pmmClient *pmm.Client
	logger    *zap.Logger
	config    *config.Config
}

func NewCollector(pmmClient *pmm.Client, config *config.Config, logger *zap.Logger) (*Collector, error) {
	if pmmClient == nil {
		return nil, fmt.Errorf("PMM client cannot be nil")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	return &Collector{
		pmmClient: pmmClient,
		logger:    logger,
		config:    config,
	}, nil
}

func (c *Collector) CollectMetrics(ctx context.Context) ([]models.Metric, error) {
	c.logger.Info("Collecting metrics from PMM")

	metrics, err := c.pmmClient.GetGoldenSignalMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Golden Signal metrics: %w", err)
	}

	c.logger.Info("Collected metrics from PMM",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Collector) CollectLatencyMetrics(ctx context.Context) ([]models.Metric, error) {
	c.logger.Debug("Collecting latency metrics")

	metrics, err := c.pmmClient.GetLatencyMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect latency metrics: %w", err)
	}

	c.logger.Debug("Collected latency metrics",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Collector) CollectTrafficMetrics(ctx context.Context) ([]models.Metric, error) {
	c.logger.Debug("Collecting traffic metrics")

	metrics, err := c.pmmClient.GetTrafficMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect traffic metrics: %w", err)
	}

	c.logger.Debug("Collected traffic metrics",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Collector) CollectErrorMetrics(ctx context.Context) ([]models.Metric, error) {
	c.logger.Debug("Collecting error metrics")

	metrics, err := c.pmmClient.GetErrorMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect error metrics: %w", err)
	}

	c.logger.Debug("Collected error metrics",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Collector) CollectSaturationMetrics(ctx context.Context) ([]models.Metric, error) {
	c.logger.Debug("Collecting saturation metrics")

	metrics, err := c.pmmClient.GetSaturationMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect saturation metrics: %w", err)
	}

	c.logger.Debug("Collected saturation metrics",
		zap.Int("count", len(metrics)))

	return metrics, nil
}

func (c *Collector) CollectMetricsWithInterval(ctx context.Context, interval time.Duration, callback func([]models.Metric)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			metrics, err := c.CollectMetrics(ctx)
			if err != nil {
				c.logger.Error("Failed to collect metrics",
					zap.Error(err))
				continue
			}

			if callback != nil {
				callback(metrics)
			}
		}
	}
}
