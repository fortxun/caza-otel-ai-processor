package analysis

import (
	"math"
	"time"
)

func (ad *AnomalyDetector) applySeasonalityAdjustment(value float64, timestamp time.Time, baseline *Baseline) float64 {
	if baseline.DataPoints < ad.config.MinDataPoints {
		return value
	}

	hourOfDay := timestamp.Hour()
	dayOfWeek := int(timestamp.Weekday())
	
	adjustedValue := value
	
	if hourPattern, exists := baseline.DailyPattern[hourOfDay]; exists {
		adjustedValue = value - (hourPattern - baseline.Mean)
	}
	
	if weekPattern, exists := baseline.WeeklyPattern[dayOfWeek]; exists {
		adjustedValue = adjustedValue - 0.5*(weekPattern - baseline.Mean)
	}
	
	return adjustedValue
}

func (ad *AnomalyDetector) getDetectionMethod(baseline *Baseline) StatisticalMethod {
	if baseline.MAD > 0.0001 && len(baseline.Values) >= ad.config.MinDataPoints {
		return MethodMAD
	}
	
	return MethodZScore
}

func calculateMAD(values []float64, median float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - median)
	}
	
	return calculateMedian(deviations)
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	valuesCopy := make([]float64, len(values))
	copy(valuesCopy, values)
	
	sort.Float64s(valuesCopy)
	
	n := len(valuesCopy)
	if n%2 == 0 {
		return (valuesCopy[n/2-1] + valuesCopy[n/2]) / 2
	}
	
	return valuesCopy[n/2]
}
