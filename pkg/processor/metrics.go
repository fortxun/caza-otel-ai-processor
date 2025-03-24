//go:build fullwasm
// +build fullwasm

// This file contains the full implementation of the metrics processor with WASM support

package processor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
)

type fullMetricsProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Metrics
	wasmRuntime  *runtime.WasmRuntime
}

func newMetricsProcessor(
	logger *zap.Logger,
	config *Config,
	nextConsumer consumer.Metrics,
) (metricsProcessor, error) {
	// Initialize WASM runtime
	wasmRuntime, err := runtime.NewWasmRuntime(logger, &runtime.WasmRuntimeConfig{
		ErrorClassifierPath:   config.Models.ErrorClassifier.Path,
		ErrorClassifierMemory: config.Models.ErrorClassifier.MemoryLimitMB,
		SamplerPath:           config.Models.ImportanceSampler.Path,
		SamplerMemory:         config.Models.ImportanceSampler.MemoryLimitMB,
		EntityExtractorPath:   config.Models.EntityExtractor.Path,
		EntityExtractorMemory: config.Models.EntityExtractor.MemoryLimitMB,
	})
	if err != nil {
		return nil, err
	}

	return &fullMetricsProcessor{
		logger:       logger,
		config:       config,
		nextConsumer: nextConsumer,
		wasmRuntime:  wasmRuntime,
	}, nil
}

func (p *fullMetricsProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// If no AI features are enabled, pass through the data unchanged
	if !p.config.Features.ErrorClassification && 
	   !p.config.Features.SmartSampling && 
	   !p.config.Features.EntityExtraction {
		return md, nil
	}

	// Use parallel processing if enabled
	if p.config.Processing.EnableParallelProcessing {
		return p.processMetricsParallel(ctx, md)
	}

	// Serial processing
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		sms := rm.ScopeMetrics()
		
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			metrics := sm.Metrics()
			
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				p.processMetric(ctx, metric, rm.Resource())
			}
		}
	}

	return md, nil
}

// Process metrics in parallel for better performance
func (p *fullMetricsProcessor) processMetricsParallel(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// Create a worker pool
	numWorkers := p.config.Processing.MaxParallelWorkers
	if numWorkers <= 0 {
		numWorkers = 8 // Default to 8 workers
	}
	pool := newWorkerPool(numWorkers)
	defer pool.close()

	// Process each resource metric
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		sms := rm.ScopeMetrics()
		
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			
			// Process metrics in parallel
			processMetricsInParallel(ctx, pool, sm.Metrics(), rm.Resource(), p.processMetric)
		}
	}

	// Wait for all metrics to be processed
	pool.wait()

	return md, nil
}

func (p *fullMetricsProcessor) processMetric(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource) {
	// Extract information for classification and enrichment
	metricInfo := map[string]interface{}{
		"name":        metric.Name(),
		"description": metric.Description(),
		"unit":        metric.Unit(),
		"resource":    attributesToMap(resource.Attributes()),
	}

	// Add attributes based on metric type
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		p.processGauge(ctx, metric, resource, metricInfo)
	case pmetric.MetricTypeSum:
		p.processSum(ctx, metric, resource, metricInfo)
	case pmetric.MetricTypeHistogram:
		p.processHistogram(ctx, metric, resource, metricInfo)
	case pmetric.MetricTypeSummary:
		p.processSummary(ctx, metric, resource, metricInfo)
	case pmetric.MetricTypeExponentialHistogram:
		p.processExponentialHistogram(ctx, metric, resource, metricInfo)
	}
}

func (p *fullMetricsProcessor) processGauge(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource, metricInfo map[string]interface{}) {
	gauge := metric.Gauge()
	dataPoints := gauge.DataPoints()
	
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		p.processDataPoint(ctx, metric, dp, resource, metricInfo)
	}
}

func (p *fullMetricsProcessor) processSum(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource, metricInfo map[string]interface{}) {
	sum := metric.Sum()
	dataPoints := sum.DataPoints()
	
	// Add sum-specific metadata
	metricInfo["is_monotonic"] = sum.IsMonotonic()
	metricInfo["aggregation_temporality"] = sum.AggregationTemporality().String()
	
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		p.processDataPoint(ctx, metric, dp, resource, metricInfo)
	}
}

func (p *fullMetricsProcessor) processHistogram(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource, metricInfo map[string]interface{}) {
	histogram := metric.Histogram()
	dataPoints := histogram.DataPoints()
	
	metricInfo["aggregation_temporality"] = histogram.AggregationTemporality().String()
	
	for i := 0; i < dataPoints.Len(); i++ {
		// Just log basic information since we don't process histogram data points specifically
		p.logger.Debug("Processing histogram data point", 
			zap.String("metric", metric.Name()),
			zap.Int("buckets", dataPoints.At(i).BucketCounts().Len()))
	}
}

func (p *fullMetricsProcessor) processSummary(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource, metricInfo map[string]interface{}) {
	summary := metric.Summary()
	dataPoints := summary.DataPoints()
	
	for i := 0; i < dataPoints.Len(); i++ {
		// Just log basic information since we don't process summary data points specifically
		dp := dataPoints.At(i)
		p.logger.Debug("Processing summary data point", 
			zap.String("metric", metric.Name()),
			zap.Uint64("count", dp.Count()),
			zap.Float64("sum", dp.Sum()),
			zap.Int("quantiles", dp.QuantileValues().Len()))
	}
}

func (p *fullMetricsProcessor) processExponentialHistogram(ctx context.Context, metric pmetric.Metric, resource pcommon.Resource, metricInfo map[string]interface{}) {
	histogram := metric.ExponentialHistogram()
	dataPoints := histogram.DataPoints()
	
	metricInfo["aggregation_temporality"] = histogram.AggregationTemporality().String()
	
	for i := 0; i < dataPoints.Len(); i++ {
		// Just log basic information since we don't process exponential histogram data points specifically
		dp := dataPoints.At(i)
		p.logger.Debug("Processing exponential histogram data point", 
			zap.String("metric", metric.Name()),
			zap.Uint64("count", dp.Count()),
			zap.Int("positive buckets", dp.Positive().BucketCounts().Len()),
			zap.Int("negative buckets", dp.Negative().BucketCounts().Len()))
	}
}

func (p *fullMetricsProcessor) processDataPoint(ctx context.Context, metric pmetric.Metric, dp pmetric.NumberDataPoint, resource pcommon.Resource, metricInfo map[string]interface{}) {
	// Add data point attributes to metric info
	pointInfo := make(map[string]interface{})
	for k, v := range metricInfo {
		pointInfo[k] = v
	}
	
	// Add data point attributes
	pointInfo["attributes"] = attributesToMap(dp.Attributes())
	
	// Add value based on data type
	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		pointInfo["value"] = dp.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		pointInfo["value"] = dp.DoubleValue()
	}
	
	// Extract entities if enabled
	if p.config.Features.EntityExtraction {
		p.extractEntities(ctx, metric, dp, pointInfo)
	}
}

func (p *fullMetricsProcessor) extractEntities(ctx context.Context, metric pmetric.Metric, dp pmetric.NumberDataPoint, metricInfo map[string]interface{}) {
	// Call entity extractor model
	result, err := p.wasmRuntime.ExtractEntities(ctx, metricInfo)
	if err != nil {
		p.logger.Error("Failed to extract entities from metric", zap.Error(err))
		return
	}

	// Add entity attributes to data point
	for k, v := range result {
		attrKey := p.config.Output.AttributeNamespace + k
		setAttribute(dp.Attributes(), attrKey, v)
	}
}

func (p *fullMetricsProcessor) shutdown(ctx context.Context) error {
	if p.wasmRuntime != nil {
		return p.wasmRuntime.Close()
	}
	return nil
}