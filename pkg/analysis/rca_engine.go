package analysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"strconv"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/llm"
	"github.com/fortxun/caza-otel-ai-processor/pkg/metrics"
	"github.com/fortxun/caza-otel-ai-processor/pkg/types/config"
	"github.com/fortxun/caza-otel-ai-processor/pkg/types/models"
	"go.uber.org/zap"
)

type RootCauseAnalysisEngine struct {
	logger       *zap.Logger
	config       *config.RootCauseAnalysisConfig
	llmClient    *llm.Client
	knowledgeBase *KnowledgeBase
}

type MetricCorrelation struct {
	SourceMetric string `json:"source_metric"`
	TargetMetric string `json:"target_metric"`
	Coefficient float64 `json:"coefficient"`
	TimeOffset int `json:"time_offset"`
}

func NewRootCauseAnalysisEngine(config *config.RootCauseAnalysisConfig, llmClient *llm.Client, logger *zap.Logger) (*RootCauseAnalysisEngine, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("root cause analysis is not enabled")
	}

	var kb *KnowledgeBase
	var err error

	if config.KnowledgeBaseEnabled {
		kb, err = NewKnowledgeBase(config.KnowledgeBasePath, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize knowledge base: %w", err)
		}
	}

	return &RootCauseAnalysisEngine{
		logger:       logger,
		config:       config,
		llmClient:    llmClient,
		knowledgeBase: kb,
	}, nil
}

func (rca *RootCauseAnalysisEngine) AnalyzeAnomaly(ctx context.Context, anomaly models.Anomaly, allMetrics []metrics.Metric, historicalData map[string]*metrics.TimeSeriesMetric) (*models.RootCause, error) {
	if !rca.config.Enabled {
		return nil, nil
	}

	rca.logger.Info("Analyzing anomaly for root cause",
		zap.String("metric", anomaly.MetricName),
		zap.Float64("value", anomaly.Value),
		zap.Float64("baseline", anomaly.Baseline),
		zap.String("severity", anomaly.Severity))

	anomalyID := fmt.Sprintf("%s-%d", anomaly.MetricName, anomaly.Timestamp.Unix())

	correlations, err := rca.findCorrelatedMetrics(anomaly, allMetrics, historicalData)
	if err != nil {
		return nil, fmt.Errorf("failed to find correlated metrics: %w", err)
	}

	sort.Slice(correlations, func(i, j int) bool {
		return math.Abs(correlations[i].Coefficient) > math.Abs(correlations[j].Coefficient)
	})

	maxRelated := 5
	if len(correlations) < maxRelated {
		maxRelated = len(correlations)
	}
	relatedMetrics := make([]string, maxRelated)
	for i := 0; i < maxRelated; i++ {
		relatedMetrics[i] = correlations[i].TargetMetric
	}

	var knowledgeBaseMatch bool
	var knowledgeBaseID string
	var recommendations []string

	if rca.config.KnowledgeBaseEnabled && rca.knowledgeBase != nil {
		match, id, recs := rca.knowledgeBase.FindMatch(anomaly, correlations)
		if match {
			knowledgeBaseMatch = true
			knowledgeBaseID = id
			recommendations = recs
			rca.logger.Info("Found knowledge base match",
				zap.String("anomaly", anomalyID),
				zap.String("knowledge_base_id", knowledgeBaseID))
		}
	}

	var description string
	var confidence float64

	if !knowledgeBaseMatch || rca.config.UseLLM {
		if rca.llmClient != nil {
			llmResult, err := rca.performLLMAnalysis(ctx, anomaly, correlations, allMetrics)
			if err != nil {
				rca.logger.Warn("Failed to perform LLM analysis",
					zap.String("anomaly", anomalyID),
					zap.Error(err))
			} else {
				description = llmResult.Description
				confidence = llmResult.Confidence
				
				if len(recommendations) == 0 {
					recommendations = llmResult.Recommendations
				}
			}
		}
	}

	if description == "" {
		description = rca.generateBasicDescription(anomaly, correlations)
		confidence = 0.5 // Medium confidence for basic description
	}

	maxRecommendations := rca.config.MaxRecommendations
	if maxRecommendations <= 0 {
		maxRecommendations = 3
	}
	if len(recommendations) > maxRecommendations {
		recommendations = recommendations[:maxRecommendations]
	}

	rootCause := &models.RootCause{
		AnomalyID:          anomalyID,
		Description:        description,
		Confidence:         confidence,
		RelatedMetrics:     relatedMetrics,
		Recommendations:    recommendations,
		KnowledgeBaseMatch: knowledgeBaseMatch,
		KnowledgeBaseID:    knowledgeBaseID,
		Timestamp:          time.Now(),
	}

	return rootCause, nil
}

func (rca *RootCauseAnalysisEngine) findCorrelatedMetrics(anomaly models.Anomaly, allMetrics []metrics.Metric, historicalData map[string]*metrics.TimeSeriesMetric) ([]MetricCorrelation, error) {
	correlations := make([]MetricCorrelation, 0)

	anomalyHistory, ok := historicalData[anomaly.MetricName]
	if !ok {
		return correlations, fmt.Errorf("no historical data for anomaly metric: %s", anomaly.MetricName)
	}

	for _, metric := range allMetrics {
		if metric.Name == anomaly.MetricName {
			continue
		}

		targetHistory, ok := historicalData[metric.Name]
		if !ok {
			continue
		}

		coefficient, timeOffset := rca.calculateCorrelation(anomalyHistory, targetHistory)

		if math.Abs(coefficient) >= rca.config.CorrelationThreshold {
			correlations = append(correlations, MetricCorrelation{
				SourceMetric: anomaly.MetricName,
				TargetMetric: metric.Name,
				Coefficient:  coefficient,
				TimeOffset:   timeOffset,
			})
		}
	}

	return correlations, nil
}

func (rca *RootCauseAnalysisEngine) calculateCorrelation(series1, series2 *metrics.TimeSeriesMetric) (float64, int) {
	
	n := min(len(series1.Values), len(series2.Values))
	if n < 2 {
		return 0, 0
	}

	x := make([]float64, n)
	y := make([]float64, n)

	for i := 0; i < n; i++ {
		x[i] = series1.Values[len(series1.Values)-n+i].Value
		y[i] = series2.Values[len(series2.Values)-n+i].Value
	}

	sumX := 0.0
	sumY := 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	numerator := 0.0
	denomX := 0.0
	denomY := 0.0

	for i := 0; i < n; i++ {
		xDiff := x[i] - meanX
		yDiff := y[i] - meanY
		numerator += xDiff * yDiff
		denomX += xDiff * xDiff
		denomY += yDiff * yDiff
	}

	if denomX == 0 || denomY == 0 {
		return 0, 0
	}

	coefficient := numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))

	return coefficient, 0
}

type LLMAnalysisResult struct {
	Description     string
	Confidence      float64
	Recommendations []string
}

func (rca *RootCauseAnalysisEngine) performLLMAnalysis(ctx context.Context, anomaly models.Anomaly, correlations []MetricCorrelation, allMetrics []metrics.Metric) (*LLMAnalysisResult, error) {
	if rca.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not initialized")
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Database Anomaly Analysis\n\n"))
	sb.WriteString(fmt.Sprintf("### Anomaly Details\n"))
	sb.WriteString(fmt.Sprintf("- Metric: %s\n", anomaly.MetricName))
	sb.WriteString(fmt.Sprintf("- Type: %s\n", anomaly.MetricType))
	sb.WriteString(fmt.Sprintf("- Current Value: %.2f %s\n", anomaly.Value, anomaly.Unit))
	sb.WriteString(fmt.Sprintf("- Baseline Value: %.2f %s\n", anomaly.Baseline, anomaly.Unit))
	sb.WriteString(fmt.Sprintf("- Deviation: %.2f standard deviations\n", anomaly.DeviationScore))
	sb.WriteString(fmt.Sprintf("- Severity: %s\n", anomaly.Severity))
	sb.WriteString(fmt.Sprintf("- Timestamp: %s\n\n", anomaly.Timestamp.Format(time.RFC3339)))

	sb.WriteString(fmt.Sprintf("### Correlated Metrics\n"))
	for i, corr := range correlations {
		if i >= 5 {
			break // Limit to top 5 correlations
		}
		
		var metricValue float64
		var metricUnit string
		for _, m := range allMetrics {
			if m.Name == corr.TargetMetric {
				metricValue = m.Value
				metricUnit = m.Unit
				break
			}
		}
		
		sb.WriteString(fmt.Sprintf("- %s: %.2f %s (correlation: %.2f)\n", 
			corr.TargetMetric, metricValue, metricUnit, corr.Coefficient))
	}
	sb.WriteString("\n")

	sb.WriteString("### Task\n")
	sb.WriteString("1. Analyze the anomaly and correlated metrics to determine the most likely root cause.\n")
	sb.WriteString("2. Provide a concise technical description of the root cause.\n")
	sb.WriteString("3. Rate your confidence in this analysis (0.0-1.0).\n")
	sb.WriteString("4. Suggest 2-3 specific actions to address the root cause.\n\n")
	sb.WriteString("Format your response as follows:\n")
	sb.WriteString("ROOT CAUSE: [concise technical description]\n")
	sb.WriteString("CONFIDENCE: [number between 0.0 and 1.0]\n")
	sb.WriteString("RECOMMENDATIONS:\n")
	sb.WriteString("1. [recommendation 1]\n")
	sb.WriteString("2. [recommendation 2]\n")
	sb.WriteString("3. [recommendation 3] (optional)\n")

	response, err := rca.llmClient.GenerateResponse(ctx, sb.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	result := &LLMAnalysisResult{
		Description:     "",
		Confidence:      0.5, // Default confidence
		Recommendations: make([]string, 0),
	}

	if rootCauseMatch := strings.Split(response.Content, "ROOT CAUSE:"); len(rootCauseMatch) > 1 {
		description := strings.Split(rootCauseMatch[1], "CONFIDENCE:")[0]
		result.Description = strings.TrimSpace(description)
	}

	if confidenceMatch := strings.Split(response.Content, "CONFIDENCE:"); len(confidenceMatch) > 1 {
		confidenceStr := strings.Split(confidenceMatch[1], "RECOMMENDATIONS:")[0]
		confidenceStr = strings.TrimSpace(confidenceStr)
		confidence, err := strconv.ParseFloat(confidenceStr, 64)
		if err == nil && confidence >= 0.0 && confidence <= 1.0 {
			result.Confidence = confidence
		}
	}

	if recommendationsMatch := strings.Split(response.Content, "RECOMMENDATIONS:"); len(recommendationsMatch) > 1 {
		recommendationsText := recommendationsMatch[1]
		lines := strings.Split(recommendationsText, "\n")
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || strings.HasPrefix(line, "3.") {
				recommendation := strings.TrimSpace(line[2:])
				if recommendation != "" {
					result.Recommendations = append(result.Recommendations, recommendation)
				}
			}
		}
	}

	return result, nil
}

func (rca *RootCauseAnalysisEngine) generateBasicDescription(anomaly models.Anomaly, correlations []MetricCorrelation) string {
	var description string

	switch anomaly.MetricType {
	case models.MetricTypeLatency:
		description = fmt.Sprintf("Abnormal increase in %s latency detected. ", anomaly.MetricName)
	case models.MetricTypeTraffic:
		if anomaly.Value > anomaly.Baseline {
			description = fmt.Sprintf("Unusual spike in %s traffic detected. ", anomaly.MetricName)
		} else {
			description = fmt.Sprintf("Unusual drop in %s traffic detected. ", anomaly.MetricName)
		}
	case models.MetricTypeError:
		description = fmt.Sprintf("Elevated error rate detected in %s. ", anomaly.MetricName)
	case models.MetricTypeSaturation:
		description = fmt.Sprintf("Resource saturation detected in %s. ", anomaly.MetricName)
	default:
		description = fmt.Sprintf("Anomaly detected in %s. ", anomaly.MetricName)
	}

	if len(correlations) > 0 {
		topCorrelation := correlations[0]
		if topCorrelation.Coefficient > 0 {
			description += fmt.Sprintf("Strongly correlated with increases in %s.", topCorrelation.TargetMetric)
		} else {
			description += fmt.Sprintf("Strongly correlated with decreases in %s.", topCorrelation.TargetMetric)
		}
	}

	return description
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
