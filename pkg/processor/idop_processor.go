package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/analysis"
	"github.com/fortxun/caza-otel-ai-processor/pkg/llm"
	"github.com/fortxun/caza-otel-ai-processor/pkg/metrics"
	"github.com/fortxun/caza-otel-ai-processor/pkg/pmm"
	"github.com/fortxun/caza-otel-ai-processor/pkg/reporting"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type IDOPProcessor struct {
	logger           *zap.Logger
	config           *Config
	pmmClient        *pmm.Client
	metricsCollector *metrics.Collector
	anomalyDetector  *analysis.AnomalyDetector
	llmClient        *llm.Client
	rcaEngine        *analysis.RootCauseAnalysisEngine
	reportGenerator  *reporting.ReportGenerator
	
	metrics          map[string]metrics.Metric
	historicalData   map[string]*metrics.TimeSeriesMetric
	anomalies        []analysis.Anomaly
	rootCauses       map[string]analysis.RootCause
	
	mutex            sync.RWMutex
	processingTicker *time.Ticker
	stopChan         chan struct{}
}

func NewIDOPProcessor(logger *zap.Logger, config *Config) (*IDOPProcessor, error) {
	if !config.PMM.Enabled {
		return nil, fmt.Errorf("PMM integration must be enabled for IDOP processor")
	}

	timeout, err := time.ParseDuration(config.PMM.Timeout)
	if err != nil {
		timeout = 30 * time.Second
		logger.Warn("Invalid PMM timeout format, using default",
			zap.String("timeout", config.PMM.Timeout),
			zap.Duration("default", timeout))
	}

	maxRetryTime, err := time.ParseDuration(config.PMM.MaxRetryTime)
	if err != nil {
		maxRetryTime = 30 * time.Second
		logger.Warn("Invalid PMM max retry time format, using default",
			zap.String("max_retry_time", config.PMM.MaxRetryTime),
			zap.Duration("default", maxRetryTime))
	}

	cacheTTL, err := time.ParseDuration(config.PMM.CacheTTL)
	if err != nil {
		cacheTTL = 5 * time.Minute
		logger.Warn("Invalid PMM cache TTL format, using default",
			zap.String("cache_ttl", config.PMM.CacheTTL),
			zap.Duration("default", cacheTTL))
	}

	queryInterval, err := time.ParseDuration(config.PMM.QueryInterval)
	if err != nil {
		queryInterval = 1 * time.Minute
		logger.Warn("Invalid PMM query interval format, using default",
			zap.String("query_interval", config.PMM.QueryInterval),
			zap.Duration("default", queryInterval))
	}

	pmmConfig := &pmm.Config{
		BaseURL:      config.PMM.BaseURL,
		APIKey:       config.PMM.APIKey,
		Timeout:      timeout,
		RetryCount:   config.PMM.RetryCount,
		MaxRetryTime: maxRetryTime,
		CacheTTL:     cacheTTL,
		CacheSize:    config.PMM.CacheSize,
	}

	pmmClient, err := pmm.NewClient(pmmConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create PMM client: %w", err)
	}

	metricQueries := make(map[string]metrics.MetricQuery)
	for name, query := range config.PMM.MetricsQueries {
		metricType := metrics.MetricTypeLatency
		metricQueries[name] = metrics.MetricQuery{
			Query: query,
			Type:  metricType,
			Unit:  "unknown", // Default unit
		}
	}

	if len(metricQueries) == 0 {
		metricQueries = metrics.DefaultDatabaseMetrics()
	}

	collectorConfig := &metrics.CollectorConfig{
		MetricsQueries: metricQueries,
		QueryInterval:  queryInterval,
		DefaultLabels:  map[string]string{"source": "idop"},
	}

	metricsCollector := metrics.NewCollector(pmmClient, collectorConfig, logger)

	anomalyDetector := analysis.NewAnomalyDetector(&config.AnomalyDetection, logger)

	var llmClient *llm.Client
	if config.LLM.Enabled {
		llmClient, err = llm.NewClient(&config.LLM, logger)
		if err != nil {
			logger.Warn("Failed to initialize LLM client, root cause analysis will use basic mode",
				zap.Error(err))
		}
	}

	var rcaEngine *analysis.RootCauseAnalysisEngine
	if config.RootCauseAnalysis.Enabled {
		rcaEngine, err = analysis.NewRootCauseAnalysisEngine(&config.RootCauseAnalysis, llmClient, logger)
		if err != nil {
			logger.Warn("Failed to initialize root cause analysis engine, will not perform RCA",
				zap.Error(err))
		}
	}

	reportGenerator := reporting.NewReportGenerator(logger)

	return &IDOPProcessor{
		logger:           logger,
		config:           config,
		pmmClient:        pmmClient,
		metricsCollector: metricsCollector,
		anomalyDetector:  anomalyDetector,
		llmClient:        llmClient,
		rcaEngine:        rcaEngine,
		reportGenerator:  reportGenerator,
		metrics:          make(map[string]metrics.Metric),
		historicalData:   make(map[string]*metrics.TimeSeriesMetric),
		anomalies:        make([]analysis.Anomaly, 0),
		rootCauses:       make(map[string]analysis.RootCause),
		stopChan:         make(chan struct{}),
	}, nil
}

func (p *IDOPProcessor) Start(ctx context.Context) error {
	if err := p.metricsCollector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	processingInterval, err := time.ParseDuration(p.config.PMM.QueryInterval)
	if err != nil {
		processingInterval = 1 * time.Minute
	}
	
	p.processingTicker = time.NewTicker(processingInterval)
	
	go func() {
		p.processData(ctx)
		
		for {
			select {
			case <-p.processingTicker.C:
				p.processData(ctx)
			case <-p.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (p *IDOPProcessor) Stop() {
	if p.processingTicker != nil {
		p.processingTicker.Stop()
	}
	close(p.stopChan)
	p.metricsCollector.Stop()
}

func (p *IDOPProcessor) processData(ctx context.Context) {
	p.logger.Debug("Starting IDOP processing cycle")
	
	allMetrics := p.metricsCollector.GetAllMetrics()
	
	p.updateMetricsState(allMetrics)
	
	if p.config.AnomalyDetection.Enabled && p.anomalyDetector != nil {
		anomalies, err := p.anomalyDetector.DetectAnomalies(ctx, allMetrics)
		if err != nil {
			p.logger.Error("Failed to detect anomalies", zap.Error(err))
		} else {
			p.mutex.Lock()
			p.anomalies = anomalies
			p.mutex.Unlock()
			
			p.logger.Info("Detected anomalies", zap.Int("count", len(anomalies)))
			
			if p.config.RootCauseAnalysis.Enabled && p.rcaEngine != nil {
				p.performRootCauseAnalysis(ctx, anomalies, allMetrics)
			}
		}
	}
	
	p.mutex.RLock()
	anomalyCount := len(p.anomalies)
	rootCauseCount := len(p.rootCauses)
	p.mutex.RUnlock()
	
	if anomalyCount > 0 {
		p.generateReport(ctx)
	}
	
	p.logger.Debug("Completed IDOP processing cycle",
		zap.Int("metrics", len(allMetrics)),
		zap.Int("anomalies", anomalyCount),
		zap.Int("root_causes", rootCauseCount))
}

func (p *IDOPProcessor) updateMetricsState(newMetrics []metrics.Metric) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	for _, metric := range newMetrics {
		p.metrics[metric.Name] = metric
		
		timeSeries, exists := p.historicalData[metric.Name]
		if !exists {
			timeSeries = &metrics.TimeSeriesMetric{
				Name:   metric.Name,
				Type:   metric.Type,
				Values: make([]metrics.TimeSeriesPoint, 0, 100),
				Labels: metric.Labels,
				Unit:   metric.Unit,
			}
			p.historicalData[metric.Name] = timeSeries
		}
		
		timeSeries.Values = append(timeSeries.Values, metrics.TimeSeriesPoint{
			Timestamp: metric.Timestamp,
			Value:     metric.Value,
		})
		
		if len(timeSeries.Values) > 100 {
			timeSeries.Values = timeSeries.Values[len(timeSeries.Values)-100:]
		}
	}
}

func (p *IDOPProcessor) performRootCauseAnalysis(ctx context.Context, anomalies []analysis.Anomaly, allMetrics []metrics.Metric) {
	p.mutex.RLock()
	historicalData := p.historicalData
	p.mutex.RUnlock()
	
	for _, anomaly := range anomalies {
		rootCause, err := p.rcaEngine.AnalyzeAnomaly(ctx, anomaly, allMetrics, historicalData)
		if err != nil {
			p.logger.Error("Failed to analyze anomaly",
				zap.String("metric", anomaly.MetricName),
				zap.Error(err))
			continue
		}
		
		if rootCause != nil {
			p.mutex.Lock()
			p.rootCauses[rootCause.AnomalyID] = *rootCause
			p.mutex.Unlock()
			
			p.logger.Info("Identified root cause",
				zap.String("anomaly", anomaly.MetricName),
				zap.String("description", rootCause.Description),
				zap.Float64("confidence", rootCause.Confidence))
		}
	}
}

func (p *IDOPProcessor) generateReport(ctx context.Context) {
	p.mutex.RLock()
	anomalies := p.anomalies
	rootCauses := make([]analysis.RootCause, 0, len(p.rootCauses))
	for _, rc := range p.rootCauses {
		rootCauses = append(rootCauses, rc)
	}
	
	relevantMetrics := make([]metrics.Metric, 0)
	for _, anomaly := range anomalies {
		if metric, exists := p.metrics[anomaly.MetricName]; exists {
			relevantMetrics = append(relevantMetrics, metric)
		}
	}
	p.mutex.RUnlock()
	
	report := p.reportGenerator.GenerateReport(anomalies, rootCauses, relevantMetrics)
	
	p.logger.Info("Generated IDOP report",
		zap.String("report_id", report.ID),
		zap.String("severity", report.OverallSeverity),
		zap.Int("anomalies", len(report.Anomalies)),
		zap.Int("root_causes", len(report.RootCauses)))
	
	reportText := p.reportGenerator.FormatText(report)
	p.logger.Info("IDOP Report", zap.String("report", reportText))
}

func (p *IDOPProcessor) EnrichMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if !p.config.PMM.Enabled {
		return md, nil
	}
	
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		sms := rm.ScopeMetrics()
		
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			metrics := sm.Metrics()
			
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				p.enrichMetric(metric, rm.Resource())
			}
		}
	}
	
	return md, nil
}

func (p *IDOPProcessor) enrichMetric(metric pmetric.Metric, resource pcommon.Resource) {
	metricName := metric.Name()
	
	var matchingAnomaly *analysis.Anomaly
	for _, anomaly := range p.anomalies {
		if anomaly.MetricName == metricName {
			matchingAnomaly = &anomaly
			break
		}
	}
	
	if matchingAnomaly == nil {
		return
	}
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		p.enrichGauge(metric, matchingAnomaly)
	case pmetric.MetricTypeSum:
		p.enrichSum(metric, matchingAnomaly)
	}
}

func (p *IDOPProcessor) enrichGauge(metric pmetric.Metric, anomaly *analysis.Anomaly) {
	gauge := metric.Gauge()
	dataPoints := gauge.DataPoints()
	
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		p.addAnomalyAttributes(dp.Attributes(), anomaly)
	}
}

func (p *IDOPProcessor) enrichSum(metric pmetric.Metric, anomaly *analysis.Anomaly) {
	sum := metric.Sum()
	dataPoints := sum.DataPoints()
	
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		p.addAnomalyAttributes(dp.Attributes(), anomaly)
	}
}

func (p *IDOPProcessor) addAnomalyAttributes(attributes pcommon.Map, anomaly *analysis.Anomaly) {
	ns := p.config.Output.AttributeNamespace
	attributes.PutBool(ns+"anomaly.detected", true)
	attributes.PutStr(ns+"anomaly.severity", anomaly.Severity)
	attributes.PutDouble(ns+"anomaly.score", anomaly.DeviationScore)
	attributes.PutDouble(ns+"anomaly.baseline", anomaly.Baseline)
	
	anomalyID := fmt.Sprintf("%s-%d", anomaly.MetricName, anomaly.Timestamp.Unix())
	if rootCause, exists := p.rootCauses[anomalyID]; exists {
		attributes.PutStr(ns+"rootcause.description", rootCause.Description)
		attributes.PutDouble(ns+"rootcause.confidence", rootCause.Confidence)
		
		if len(rootCause.Recommendations) > 0 {
			attributes.PutStr(ns+"rootcause.recommendations", strings.Join(rootCause.Recommendations, "; "))
		}
	}
}
