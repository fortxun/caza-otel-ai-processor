package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/fortxun/caza-otel-ai-processor/pkg/metrics"
	"go.uber.org/zap"
)

type CorrelationAnalyzer struct {
	logger *zap.Logger
	config *CorrelationConfig
}

type CorrelationConfig struct {
	Methods []string `json:"methods"`
	
	MinCorrelation float64 `json:"min_correlation"`
	
	LagWindow int `json:"lag_window"`
	
	MinDataPoints int `json:"min_data_points"`
}

type CorrelationResult struct {
	SourceMetric string `json:"source_metric"`
	
	TargetMetric string `json:"target_metric"`
	
	Coefficient float64 `json:"coefficient"`
	
	Method string `json:"method"`
	
	TimeOffset int `json:"time_offset"`
	
	Direction int `json:"direction"`
	
	Significance float64 `json:"significance"`
}

func NewCorrelationAnalyzer(config *CorrelationConfig, logger *zap.Logger) *CorrelationAnalyzer {
	if config == nil {
		config = &CorrelationConfig{
			Methods:        []string{"pearson"},
			MinCorrelation: 0.7,
			LagWindow:      5,
			MinDataPoints:  10,
		}
	}
	
	return &CorrelationAnalyzer{
		logger: logger,
		config: config,
	}
}

func (ca *CorrelationAnalyzer) AnalyzeCorrelations(
	sourceMetric string,
	historicalData map[string]*metrics.TimeSeriesMetric,
) ([]CorrelationResult, error) {
	results := make([]CorrelationResult, 0)
	
	sourceSeries, exists := historicalData[sourceMetric]
	if !exists {
		return nil, fmt.Errorf("source metric not found: %s", sourceMetric)
	}
	
	if len(sourceSeries.Values) < ca.config.MinDataPoints {
		return nil, fmt.Errorf("not enough data points for source metric: %s", sourceMetric)
	}
	
	for targetMetric, targetSeries := range historicalData {
		if targetMetric == sourceMetric {
			continue
		}
		
		if len(targetSeries.Values) < ca.config.MinDataPoints {
			continue
		}
		
		for _, method := range ca.config.Methods {
			switch method {
			case "pearson":
				result, err := ca.calculatePearsonCorrelation(sourceSeries, targetSeries)
				if err != nil {
					ca.logger.Debug("Failed to calculate Pearson correlation",
						zap.String("source", sourceMetric),
						zap.String("target", targetMetric),
						zap.Error(err))
					continue
				}
				
				if math.Abs(result.Coefficient) >= ca.config.MinCorrelation {
					results = append(results, result)
				}
				
			case "spearman":
				result, err := ca.calculateSpearmanCorrelation(sourceSeries, targetSeries)
				if err != nil {
					ca.logger.Debug("Failed to calculate Spearman correlation",
						zap.String("source", sourceMetric),
						zap.String("target", targetMetric),
						zap.Error(err))
					continue
				}
				
				if math.Abs(result.Coefficient) >= ca.config.MinCorrelation {
					results = append(results, result)
				}
				
			case "lagged":
				lagResults, err := ca.calculateLaggedCorrelation(sourceSeries, targetSeries)
				if err != nil {
					ca.logger.Debug("Failed to calculate lagged correlation",
						zap.String("source", sourceMetric),
						zap.String("target", targetMetric),
						zap.Error(err))
					continue
				}
				
				for _, result := range lagResults {
					if math.Abs(result.Coefficient) >= ca.config.MinCorrelation {
						results = append(results, result)
					}
				}
			}
		}
	}
	
	sort.Slice(results, func(i, j int) bool {
		return math.Abs(results[i].Coefficient) > math.Abs(results[j].Coefficient)
	})
	
	return results, nil
}

func (ca *CorrelationAnalyzer) calculatePearsonCorrelation(
	series1 *metrics.TimeSeriesMetric,
	series2 *metrics.TimeSeriesMetric,
) (CorrelationResult, error) {
	result := CorrelationResult{
		SourceMetric: series1.Name,
		TargetMetric: series2.Name,
		Method:       "pearson",
		TimeOffset:   0,
		Direction:    0,
		Significance: 1.0,
	}
	
	s1Start := series1.Values[0].Timestamp
	s1End := series1.Values[len(series1.Values)-1].Timestamp
	s2Start := series2.Values[0].Timestamp
	s2End := series2.Values[len(series2.Values)-1].Timestamp
	
	start := s1Start
	if s2Start.After(start) {
		start = s2Start
	}
	
	end := s1End
	if s2End.Before(end) {
		end = s2End
	}
	
	var x, y []float64
	
	for _, point := range series1.Values {
		if !point.Timestamp.Before(start) && !point.Timestamp.After(end) {
			x = append(x, point.Value)
		}
	}
	
	for _, point := range series2.Values {
		if !point.Timestamp.Before(start) && !point.Timestamp.After(end) {
			y = append(y, point.Value)
		}
	}
	
	n := min(len(x), len(y))
	if n < ca.config.MinDataPoints {
		return result, fmt.Errorf("not enough overlapping data points: %d", n)
	}
	
	x = x[:n]
	y = y[:n]
	
	sumX, sumY := 0.0, 0.0
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
		return result, fmt.Errorf("zero variance in one or both time series")
	}
	
	coefficient := numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))
	result.Coefficient = coefficient
	
	tValue := coefficient * math.Sqrt(float64(n-2) / (1 - coefficient*coefficient))
	result.Significance = 1.0 / (1.0 + math.Abs(tValue))
	
	return result, nil
}

func (ca *CorrelationAnalyzer) calculateSpearmanCorrelation(
	series1 *metrics.TimeSeriesMetric,
	series2 *metrics.TimeSeriesMetric,
) (CorrelationResult, error) {
	result := CorrelationResult{
		SourceMetric: series1.Name,
		TargetMetric: series2.Name,
		Method:       "spearman",
		TimeOffset:   0,
		Direction:    0,
		Significance: 1.0,
	}
	
	s1Start := series1.Values[0].Timestamp
	s1End := series1.Values[len(series1.Values)-1].Timestamp
	s2Start := series2.Values[0].Timestamp
	s2End := series2.Values[len(series2.Values)-1].Timestamp
	
	start := s1Start
	if s2Start.After(start) {
		start = s2Start
	}
	
	end := s1End
	if s2End.Before(end) {
		end = s2End
	}
	
	var x, y []float64
	
	for _, point := range series1.Values {
		if !point.Timestamp.Before(start) && !point.Timestamp.After(end) {
			x = append(x, point.Value)
		}
	}
	
	for _, point := range series2.Values {
		if !point.Timestamp.Before(start) && !point.Timestamp.After(end) {
			y = append(y, point.Value)
		}
	}
	
	n := min(len(x), len(y))
	if n < ca.config.MinDataPoints {
		return result, fmt.Errorf("not enough overlapping data points: %d", n)
	}
	
	x = x[:n]
	y = y[:n]
	
	xRanks := rankValues(x)
	yRanks := rankValues(y)
	
	sumX, sumY := 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += xRanks[i]
		sumY += yRanks[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)
	
	numerator := 0.0
	denomX := 0.0
	denomY := 0.0
	
	for i := 0; i < n; i++ {
		xDiff := xRanks[i] - meanX
		yDiff := yRanks[i] - meanY
		numerator += xDiff * yDiff
		denomX += xDiff * xDiff
		denomY += yDiff * yDiff
	}
	
	if denomX == 0 || denomY == 0 {
		return result, fmt.Errorf("zero variance in one or both rank series")
	}
	
	coefficient := numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))
	result.Coefficient = coefficient
	
	tValue := coefficient * math.Sqrt(float64(n-2) / (1 - coefficient*coefficient))
	result.Significance = 1.0 / (1.0 + math.Abs(tValue))
	
	return result, nil
}

func (ca *CorrelationAnalyzer) calculateLaggedCorrelation(
	series1 *metrics.TimeSeriesMetric,
	series2 *metrics.TimeSeriesMetric,
) ([]CorrelationResult, error) {
	results := make([]CorrelationResult, 0)
	
	x := make([]float64, len(series1.Values))
	for i, point := range series1.Values {
		x[i] = point.Value
	}
	
	y := make([]float64, len(series2.Values))
	for i, point := range series2.Values {
		y[i] = point.Value
	}
	
	if len(x) < ca.config.MinDataPoints || len(y) < ca.config.MinDataPoints {
		return results, fmt.Errorf("not enough data points for lagged correlation")
	}
	
	maxLag := ca.config.LagWindow
	if maxLag > len(x)/3 {
		maxLag = len(x) / 3 // Limit lag to 1/3 of the series length
	}
	
	for lag := 1; lag <= maxLag; lag++ {
		if len(x) <= lag || len(y) <= lag {
			continue
		}
		
		xLagged := x[:len(x)-lag]
		yLagged := y[lag:]
		
		n := min(len(xLagged), len(yLagged))
		if n < ca.config.MinDataPoints {
			continue
		}
		
		xLagged = xLagged[:n]
		yLagged = yLagged[:n]
		
		sumX, sumY := 0.0, 0.0
		for i := 0; i < n; i++ {
			sumX += xLagged[i]
			sumY += yLagged[i]
		}
		meanX := sumX / float64(n)
		meanY := sumY / float64(n)
		
		numerator := 0.0
		denomX := 0.0
		denomY := 0.0
		
		for i := 0; i < n; i++ {
			xDiff := xLagged[i] - meanX
			yDiff := yLagged[i] - meanY
			numerator += xDiff * yDiff
			denomX += xDiff * xDiff
			denomY += yDiff * yDiff
		}
		
		if denomX == 0 || denomY == 0 {
			continue
		}
		
		coefficient := numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))
		
		tValue := coefficient * math.Sqrt(float64(n-2) / (1 - coefficient*coefficient))
		significance := 1.0 / (1.0 + math.Abs(tValue))
		
		results = append(results, CorrelationResult{
			SourceMetric: series1.Name,
			TargetMetric: series2.Name,
			Coefficient:  coefficient,
			Method:       "lagged",
			TimeOffset:   lag,
			Direction:    1, // Source leads target
			Significance: significance,
		})
	}
	
	for lag := 1; lag <= maxLag; lag++ {
		if len(x) <= lag || len(y) <= lag {
			continue
		}
		
		xLagged := x[lag:]
		yLagged := y[:len(y)-lag]
		
		n := min(len(xLagged), len(yLagged))
		if n < ca.config.MinDataPoints {
			continue
		}
		
		xLagged = xLagged[:n]
		yLagged = yLagged[:n]
		
		sumX, sumY := 0.0, 0.0
		for i := 0; i < n; i++ {
			sumX += xLagged[i]
			sumY += yLagged[i]
		}
		meanX := sumX / float64(n)
		meanY := sumY / float64(n)
		
		numerator := 0.0
		denomX := 0.0
		denomY := 0.0
		
		for i := 0; i < n; i++ {
			xDiff := xLagged[i] - meanX
			yDiff := yLagged[i] - meanY
			numerator += xDiff * yDiff
			denomX += xDiff * xDiff
			denomY += yDiff * yDiff
		}
		
		if denomX == 0 || denomY == 0 {
			continue
		}
		
		coefficient := numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))
		
		tValue := coefficient * math.Sqrt(float64(n-2) / (1 - coefficient*coefficient))
		significance := 1.0 / (1.0 + math.Abs(tValue))
		
		results = append(results, CorrelationResult{
			SourceMetric: series1.Name,
			TargetMetric: series2.Name,
			Coefficient:  coefficient,
			Method:       "lagged",
			TimeOffset:   -lag, // Negative lag means target leads source
			Direction:    -1,   // Target leads source
			Significance: significance,
		})
	}
	
	sort.Slice(results, func(i, j int) bool {
		return math.Abs(results[i].Coefficient) > math.Abs(results[j].Coefficient)
	})
	
	return results, nil
}

func rankValues(values []float64) []float64 {
	type indexedValue struct {
		index int
		value float64
	}
	
	indexed := make([]indexedValue, len(values))
	for i, v := range values {
		indexed[i] = indexedValue{i, v}
	}
	
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].value < indexed[j].value
	})
	
	ranks := make([]float64, len(values))
	for i := 0; i < len(indexed); {
		j := i
		for j < len(indexed)-1 && indexed[j].value == indexed[j+1].value {
			j++
		}
		
		if j > i {
			avgRank := float64(i+j) / 2.0 + 1.0
			for k := i; k <= j; k++ {
				ranks[indexed[k].index] = avgRank
			}
			i = j + 1
		} else {
			ranks[indexed[i].index] = float64(i) + 1.0
			i++
		}
	}
	
	return ranks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
