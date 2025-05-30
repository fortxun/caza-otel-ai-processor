package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fortxun/idop/pkg/alerting"
	"github.com/fortxun/idop/pkg/analysis"
	"github.com/fortxun/idop/pkg/metrics"
	"github.com/fortxun/idop/pkg/reporting"
	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type IDOPProcessor struct {
	config      *config.Config
	collector   *metrics.Collector
	detector    *analysis.AnomalyDetector
	reporter    *reporting.ReportGenerator
	notifier    *alerting.Notifier
	logger      *zap.Logger
	stopCh      chan struct{}
	wg          sync.WaitGroup
	interval    time.Duration
	lastReport  time.Time
	reportMutex sync.Mutex
}

func NewIDOPProcessor(
	config *config.Config,
	collector *metrics.Collector,
	detector *analysis.AnomalyDetector,
	reporter *reporting.ReportGenerator,
	notifier *alerting.Notifier,
	logger *zap.Logger,
) (*IDOPProcessor, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if collector == nil {
		return nil, fmt.Errorf("collector cannot be nil")
	}

	if detector == nil {
		return nil, fmt.Errorf("detector cannot be nil")
	}

	if reporter == nil {
		return nil, fmt.Errorf("reporter cannot be nil")
	}

	if notifier == nil {
		return nil, fmt.Errorf("notifier cannot be nil")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	interval := 1 * time.Minute // Default interval
	if config.Server.MetricsInterval != "" {
		var err error
		interval, err = time.ParseDuration(config.Server.MetricsInterval)
		if err != nil {
			logger.Warn("Invalid metrics interval format, using default",
				zap.String("interval", config.Server.MetricsInterval),
				zap.Error(err))
		}
	}

	return &IDOPProcessor{
		config:     config,
		collector:  collector,
		detector:   detector,
		reporter:   reporter,
		notifier:   notifier,
		logger:     logger,
		stopCh:     make(chan struct{}),
		interval:   interval,
		lastReport: time.Time{},
	}, nil
}

func (p *IDOPProcessor) Start(ctx context.Context) error {
	p.logger.Info("Starting IDOP processor",
		zap.Duration("interval", p.interval))

	p.wg.Add(1)
	go p.processLoop(ctx)

	return nil
}

func (p *IDOPProcessor) Stop() error {
	p.logger.Info("Stopping IDOP processor")

	close(p.stopCh)
	p.wg.Wait()

	return nil
}

func (p *IDOPProcessor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	if err := p.ProcessMetrics(ctx); err != nil {
		p.logger.Error("Failed to process metrics",
			zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Context cancelled, stopping processor loop")
			return
		case <-p.stopCh:
			p.logger.Info("Stop signal received, stopping processor loop")
			return
		case <-ticker.C:
			if err := p.ProcessMetrics(ctx); err != nil {
				p.logger.Error("Failed to process metrics",
					zap.Error(err))
			}
		}
	}
}

func (p *IDOPProcessor) ProcessMetrics(ctx context.Context) error {
	p.logger.Debug("Processing metrics")

	metrics, err := p.collector.CollectMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	p.logger.Info("Collected metrics",
		zap.Int("count", len(metrics)))

	anomalies, err := p.detector.DetectAnomalies(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to detect anomalies: %w", err)
	}

	p.logger.Info("Detected anomalies",
		zap.Int("count", len(anomalies)))

	if len(anomalies) == 0 {
		p.logger.Debug("No anomalies detected, skipping report generation")
		return nil
	}

	rootCauses := make([]models.RootCause, 0)

	report := p.reporter.GenerateReport(anomalies, rootCauses)

	if err := p.notifier.SendAlert(ctx, report); err != nil {
		p.logger.Error("Failed to send alert",
			zap.Error(err))
	}

	p.reportMutex.Lock()
	p.lastReport = time.Now()
	p.reportMutex.Unlock()

	return nil
}

func (p *IDOPProcessor) GetLastReportTime() time.Time {
	p.reportMutex.Lock()
	defer p.reportMutex.Unlock()
	return p.lastReport
}
