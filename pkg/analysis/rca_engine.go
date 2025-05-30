package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
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
	logger             *zap.Logger
	config             *config.RootCauseAnalysisConfig
	llmClient          *llm.Client
	knowledgeBase      *KnowledgeBase
	correlationAnalyzer *CorrelationAnalyzer
	historicalAnomalies []models.Anomaly
}

type MetricCorrelation struct {
	SourceMetric string `json:"source_metric"`
	TargetMetric string `json:"target_metric"`
	Coefficient  float64 `json:"coefficient"`
	TimeOffset   int `json:"time_offset"`
	Direction    int `json:"direction"`
	Method       string `json:"method"`
}

type CausalRelationship struct {
	CauseMetric   string  `json:"cause_metric"`
	EffectMetric  string  `json:"effect_metric"`
	Strength      float64 `json:"strength"`
	Confidence    float64 `json:"confidence"`
	TimeDelay     int     `json:"time_delay"`
	DirectionType string  `json:"direction_type"` // "positive" or "negative"
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

	correlationConfig := &CorrelationConfig{
		Methods:        []string{"pearson", "spearman", "lagged"},
		MinCorrelation: config.CorrelationThreshold,
		LagWindow:      5,
		MinDataPoints:  10,
	}

	return &RootCauseAnalysisEngine{
		logger:             logger,
		config:             config,
		llmClient:          llmClient,
		knowledgeBase:      kb,
		correlationAnalyzer: NewCorrelationAnalyzer(correlationConfig, logger),
		historicalAnomalies: make([]models.Anomaly, 0, 100),
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

	rca.addHistoricalAnomaly(anomaly)

	correlationResults, err := rca.correlationAnalyzer.AnalyzeCorrelations(anomaly.MetricName, historicalData)
	if err != nil {
		rca.logger.Warn("Failed to analyze correlations, falling back to basic correlation",
			zap.String("metric", anomaly.MetricName),
			zap.Error(err))
		
		correlations, err := rca.findCorrelatedMetrics(anomaly, allMetrics, historicalData)
		if err != nil {
			return nil, fmt.Errorf("failed to find correlated metrics: %w", err)
		}
		
		return rca.analyzeWithCorrelations(ctx, anomaly, correlations, allMetrics, historicalData)
	}
	
	// Convert correlation results to metric correlations
	correlations := make([]MetricCorrelation, len(correlationResults))
	for i, result := range correlationResults {
		correlations[i] = MetricCorrelation{
			SourceMetric: result.SourceMetric,
			TargetMetric: result.TargetMetric,
			Coefficient:  result.Coefficient,
			TimeOffset:   result.TimeOffset,
			Direction:    result.Direction,
			Method:       result.Method,
		}
	}
	
	causalRelationships := rca.inferCausalRelationships(correlations, historicalData)
	
	sort.Slice(correlations, func(i, j int) bool {
		return math.Abs(correlations[i].Coefficient) > math.Abs(correlations[j].Coefficient)
	})
	
	return rca.analyzeWithCorrelationsAndCausality(ctx, anomaly, correlations, causalRelationships, allMetrics, historicalData)
}

func (rca *RootCauseAnalysisEngine) addHistoricalAnomaly(anomaly models.Anomaly) {
	rca.historicalAnomalies = append(rca.historicalAnomalies, anomaly)
	if len(rca.historicalAnomalies) > 100 {
		rca.historicalAnomalies = rca.historicalAnomalies[len(rca.historicalAnomalies)-100:]
	}
}

func (rca *RootCauseAnalysisEngine) analyzeWithCorrelations(
	ctx context.Context,
	anomaly models.Anomaly,
	correlations []MetricCorrelation,
	allMetrics []metrics.Metric,
	historicalData map[string]*metrics.TimeSeriesMetric,
) (*models.RootCause, error) {
	anomalyID := fmt.Sprintf("%s-%d", anomaly.MetricName, anomaly.Timestamp.Unix())
	
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
	var knowledgeMatches []KnowledgeEntry

	if rca.config.KnowledgeBaseEnabled && rca.knowledgeBase != nil {
		match, id, recs, entries := rca.knowledgeBase.FindMatchWithEntries(anomaly, correlations)
		if match {
			knowledgeBaseMatch = true
			knowledgeBaseID = id
			recommendations = recs
			knowledgeMatches = entries
			rca.logger.Info("Found knowledge base match",
				zap.String("anomaly", anomalyID),
				zap.String("knowledge_base_id", knowledgeBaseID))
		}
	}

	var description string
	var confidence float64

	if !knowledgeBaseMatch || rca.config.UseLLM {
		if rca.llmClient != nil {
			llmResult, err := rca.performEnhancedLLMAnalysis(ctx, anomaly, correlations, knowledgeMatches, allMetrics, historicalData)
			if err != nil {
				rca.logger.Warn("Failed to perform enhanced LLM analysis, falling back to basic analysis",
					zap.String("anomaly", anomalyID),
					zap.Error(err))
				
				basicResult, err := rca.performLLMAnalysis(ctx, anomaly, correlations, allMetrics)
				if err != nil {
					rca.logger.Warn("Failed to perform basic LLM analysis",
						zap.String("anomaly", anomalyID),
						zap.Error(err))
				} else {
					description = basicResult.Description
					confidence = basicResult.Confidence
					
					if len(recommendations) == 0 {
						recommendations = basicResult.Recommendations
					}
				}
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

func (rca *RootCauseAnalysisEngine) analyzeWithCorrelationsAndCausality(
	ctx context.Context,
	anomaly models.Anomaly,
	correlations []MetricCorrelation,
	causalRelationships []CausalRelationship,
	allMetrics []metrics.Metric,
	historicalData map[string]*metrics.TimeSeriesMetric,
) (*models.RootCause, error) {
	anomalyID := fmt.Sprintf("%s-%d", anomaly.MetricName, anomaly.Timestamp.Unix())
	
	relatedMetricsMap := make(map[string]bool)
	
	for i, corr := range correlations {
		if i >= 5 {
			break // Limit to top 5 correlations
		}
		relatedMetricsMap[corr.TargetMetric] = true
	}
	
	for _, causal := range causalRelationships {
		if causal.EffectMetric == anomaly.MetricName {
			relatedMetricsMap[causal.CauseMetric] = true
		}
	}
	
	relatedMetrics := make([]string, 0, len(relatedMetricsMap))
	for metric := range relatedMetricsMap {
		relatedMetrics = append(relatedMetrics, metric)
	}
	
	if len(relatedMetrics) > 5 {
		relatedMetrics = relatedMetrics[:5]
	}

	var knowledgeBaseMatch bool
	var knowledgeBaseID string
	var recommendations []string
	var knowledgeMatches []KnowledgeEntry

	if rca.config.KnowledgeBaseEnabled && rca.knowledgeBase != nil {
		match, id, recs, entries := rca.knowledgeBase.FindMatchWithEntries(anomaly, correlations)
		if match {
			knowledgeBaseMatch = true
			knowledgeBaseID = id
			recommendations = recs
			knowledgeMatches = entries
			rca.logger.Info("Found knowledge base match",
				zap.String("anomaly", anomalyID),
				zap.String("knowledge_base_id", knowledgeBaseID))
		}
	}

	similarAnomalies := rca.findSimilarHistoricalAnomalies(anomaly)

	var description string
	var confidence float64

	if rca.llmClient != nil {
		llmResult, err := rca.performEnhancedLLMAnalysis(
			ctx, 
			anomaly, 
			correlations, 
			knowledgeMatches, 
			allMetrics, 
			historicalData,
		)
		
		if err != nil {
			rca.logger.Warn("Failed to perform enhanced LLM analysis",
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

	if description == "" {
		if knowledgeBaseMatch {
			description = fmt.Sprintf("Based on knowledge base match: %s", knowledgeMatches[0].Description)
			confidence = 0.7 // Higher confidence for knowledge base match
		} else {
			description = rca.generateBasicDescription(anomaly, correlations)
			confidence = 0.5 // Medium confidence for basic description
		}
	}

	// If we have causal relationships, enhance the description
	if len(causalRelationships) > 0 {
		for _, causal := range causalRelationships {
			if causal.EffectMetric == anomaly.MetricName {
				description = fmt.Sprintf("%s Likely caused by changes in %s (causal strength: %.2f).", 
					description, causal.CauseMetric, causal.Strength)
				break
			}
		}
	}

	// Limit recommendations
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

func (rca *RootCauseAnalysisEngine) findSimilarHistoricalAnomalies(anomaly models.Anomaly) []models.Anomaly {
	similar := make([]models.Anomaly, 0)
	
	for _, historical := range rca.historicalAnomalies {
		if historical.MetricName == anomaly.MetricName && 
		   historical.Timestamp.Equal(anomaly.Timestamp) {
			continue
		}
		
		if historical.MetricName == anomaly.MetricName || 
		   (historical.MetricType == anomaly.MetricType && 
		    math.Abs(historical.DeviationScore-anomaly.DeviationScore) < 1.0) {
			similar = append(similar, historical)
		}
	}
	
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].Timestamp.After(similar[j].Timestamp)
	})
	
	if len(similar) > 5 {
		similar = similar[:5]
	}
	
	return similar
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
				Direction:    0, // No direction information in basic correlation
				Method:       "pearson",
			})
		}
	}

	return correlations, nil
}

func (rca *RootCauseAnalysisEngine) inferCausalRelationships(correlations []MetricCorrelation, historicalData map[string]*metrics.TimeSeriesMetric) []CausalRelationship {
	causalRelationships := make([]CausalRelationship, 0)
	
	metricGroups := make(map[string][]MetricCorrelation)
	for _, corr := range correlations {
		metricGroups[corr.SourceMetric] = append(metricGroups[corr.SourceMetric], corr)
	}
	
	for sourceMetric, correlations := range metricGroups {
		for _, corr := range correlations {
			if corr.Method == "lagged" && corr.TimeOffset != 0 {
				var causeMetric, effectMetric string
				var timeDelay int
				
				if corr.TimeOffset > 0 {
					causeMetric = corr.SourceMetric
					effectMetric = corr.TargetMetric
					timeDelay = corr.TimeOffset
				} else {
					causeMetric = corr.TargetMetric
					effectMetric = corr.SourceMetric
					timeDelay = -corr.TimeOffset
				}
				
				directionType := "positive"
				if corr.Coefficient < 0 {
					directionType = "negative"
				}
				
				strength := math.Abs(corr.Coefficient) * (1.0 - float64(timeDelay)/10.0)
				if strength < 0.1 {
					strength = 0.1
				}
				
				confidence := strength
				if corr.Method == "lagged" {
					confidence *= 1.2 // Boost confidence for lagged correlations
				}
				if confidence > 1.0 {
					confidence = 1.0
				}
				
				causalRelationships = append(causalRelationships, CausalRelationship{
					CauseMetric:   causeMetric,
					EffectMetric:  effectMetric,
					Strength:      strength,
					Confidence:    confidence,
					TimeDelay:     timeDelay,
					DirectionType: directionType,
				})
			}
		}
		
		for _, corr := range correlations {
			if corr.Direction != 0 && math.Abs(corr.Coefficient) > 0.8 {
				var causeMetric, effectMetric string
				
				if corr.Direction > 0 {
					causeMetric = corr.SourceMetric
					effectMetric = corr.TargetMetric
				} else {
					causeMetric = corr.TargetMetric
					effectMetric = corr.SourceMetric
				}
				
				directionType := "positive"
				if corr.Coefficient < 0 {
					directionType = "negative"
				}
				
				strength := math.Abs(corr.Coefficient)
				confidence := strength * 0.9 // Slightly lower confidence without time lag evidence
				
				causalRelationships = append(causalRelationships, CausalRelationship{
					CauseMetric:   causeMetric,
					EffectMetric:  effectMetric,
					Strength:      strength,
					Confidence:    confidence,
					TimeDelay:     0,
					DirectionType: directionType,
				})
			}
		}
	}
	
	sort.Slice(causalRelationships, func(i, j int) bool {
		return causalRelationships[i].Confidence > causalRelationships[j].Confidence
	})
	
	return causalRelationships
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

type LLMRootCauseResponse struct {
	RootCause       string   `json:"root_cause"`
	Explanation     string   `json:"explanation"`
	Confidence      int      `json:"confidence"`
	Recommendations []string `json:"recommendations"`
	Impact          string   `json:"impact"`
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

func (rca *RootCauseAnalysisEngine) performEnhancedLLMAnalysis(
	ctx context.Context,
	anomaly models.Anomaly,
	correlations []MetricCorrelation,
	knowledgeMatches []KnowledgeEntry,
	allMetrics []metrics.Metric,
	historicalData map[string]*metrics.TimeSeriesMetric,
) (*LLMAnalysisResult, error) {
	if rca.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not initialized")
	}
	
	// Convert correlations to the format expected by the prompt generator
	promptCorrelations := make([]analysis.CorrelationResult, len(correlations))
	for i, corr := range correlations {
		promptCorrelations[i] = analysis.CorrelationResult{
			SourceMetric: corr.SourceMetric,
			TargetMetric: corr.TargetMetric,
			Coefficient:  corr.Coefficient,
			Method:       corr.Method,
			TimeOffset:   corr.TimeOffset,
			Direction:    corr.Direction,
		}
	}
	
	similarAnomalies := rca.findSimilarHistoricalAnomalies(anomaly)
	
	prompt, err := llm.GenerateRootCauseAnalysisPrompt(
		anomaly,
		promptCorrelations,
		knowledgeMatches,
		similarAnomalies,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate enhanced prompt: %w", err)
	}
	
	response, err := rca.llmClient.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}
	
	result := &LLMAnalysisResult{
		Description:     "",
		Confidence:      0.5, // Default confidence
		Recommendations: make([]string, 0),
	}
	
	var llmResponse LLMRootCauseResponse
	
	jsonContent := response.Content
	
	startIdx := strings.Index(jsonContent, "{")
	endIdx := strings.LastIndex(jsonContent, "}")
	
	if startIdx >= 0 && endIdx > startIdx {
		jsonContent = jsonContent[startIdx:endIdx+1]
	}
	
	err = json.Unmarshal([]byte(jsonContent), &llmResponse)
	if err == nil {
		result.Description = llmResponse.RootCause
		if llmResponse.Explanation != "" {
			result.Description += ": " + llmResponse.Explanation
		}
		
		// Convert confidence from 0-100 to 0-1
		result.Confidence = float64(llmResponse.Confidence) / 100.0
		if result.Confidence < 0 {
			result.Confidence = 0
		} else if result.Confidence > 1 {
			result.Confidence = 1
		}
		
		result.Recommendations = llmResponse.Recommendations
	} else {
		rca.logger.Warn("Failed to parse JSON response, falling back to text parsing",
			zap.Error(err),
			zap.String("response", response.Content))
		
		if rootCauseMatch := strings.Split(response.Content, "root cause:"); len(rootCauseMatch) > 1 {
			description := strings.Split(rootCauseMatch[1], "confidence:")[0]
			result.Description = strings.TrimSpace(description)
		}
		
		if confidenceMatch := strings.Split(strings.ToLower(response.Content), "confidence:"); len(confidenceMatch) > 1 {
			confidenceStr := strings.Split(confidenceMatch[1], "recommendations:")[0]
			confidenceStr = strings.TrimSpace(confidenceStr)
			
			confidenceRegex := regexp.MustCompile(`(\d+)`)
			matches := confidenceRegex.FindStringSubmatch(confidenceStr)
			if len(matches) > 1 {
				confidenceVal, err := strconv.Atoi(matches[1])
				if err == nil {
					if confidenceVal > 1 {
						result.Confidence = float64(confidenceVal) / 100.0
					} else {
						result.Confidence = float64(confidenceVal)
					}
				}
			}
		}
		
		if recommendationsMatch := strings.Split(strings.ToLower(response.Content), "recommendations:"); len(recommendationsMatch) > 1 {
			recommendationsText := recommendationsMatch[1]
			lines := strings.Split(recommendationsText, "\n")
			
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || 
				   strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") || 
				   strings.HasPrefix(line, "5.") || strings.HasPrefix(line, "-") {
					var recommendation string
					if strings.HasPrefix(line, "-") {
						recommendation = strings.TrimSpace(line[1:])
					} else {
						recommendation = strings.TrimSpace(line[2:])
					}
					if recommendation != "" {
						result.Recommendations = append(result.Recommendations, recommendation)
					}
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
