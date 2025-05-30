package analysis

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/fortxun/idop/pkg/types/config"
	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type StatisticalMethod string

const (
	MethodZScore StatisticalMethod = "zscore"
	MethodMAD    StatisticalMethod = "mad"
)

type AnomalyDetector struct {
	logger    *zap.Logger
	config    *config.AnalysisConfig
	baselines map[string]*Baseline
	mutex     sync.RWMutex
}

type Baseline struct {
	Mean          float64
	StdDev        float64
	Median        float64
	MAD           float64
	Min           float64
	Max           float64
	LastUpdated   time.Time
	DataPoints    int
	Values        []float64
	DailyPattern  map[int]float64
	WeeklyPattern map[int]float64
}

func NewAnomalyDetector(config *config.AnalysisConfig, logger *zap.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		logger:    logger,
		config:    config,
		baselines: make(map[string]*Baseline),
	}
}

func (ad *AnomalyDetector) DetectAnomalies(ctx context.Context, metrics []models.Metric) ([]models.Anomaly, error) {
	if !ad.config.Enabled {
		return nil, nil
	}

	anomalies := make([]models.Anomaly, 0)

	for _, metric := range metrics {
		if math.IsNaN(metric.Value) || math.IsInf(metric.Value, 0) {
			continue
		}

		baseline, err := ad.getOrCreateBaseline(metric.Name)
		if err != nil {
			ad.logger.Warn("Failed to get baseline for metric",
				zap.String("metric", metric.Name),
				zap.Error(err))
			continue
		}

		if ad.config.AdaptiveBaseline {
			ad.updateBaseline(metric.Name, metric.Value, metric.Timestamp)
		}

		if baseline.DataPoints < ad.config.MinDataPoints {
			ad.logger.Debug("Not enough data points for anomaly detection",
				zap.String("metric", metric.Name),
				zap.Int("data_points", baseline.DataPoints),
				zap.Int("required", ad.config.MinDataPoints))
			continue
		}

		var adjustedValue float64
		if ad.config.SeasonalityAdjust {
			adjustedValue = ad.applySeasonalityAdjustment(metric.Value, metric.Timestamp, baseline)
		} else {
			adjustedValue = metric.Value
		}

		var deviation float64
		var anomalyDetected bool
		detectionMethod := ad.getDetectionMethod(baseline)

		if detectionMethod == MethodZScore {
			deviation = (adjustedValue - baseline.Mean) / baseline.StdDev
			if math.IsNaN(deviation) || math.IsInf(deviation, 0) {
				if baseline.StdDev < 0.0001 {
					deviation = math.Abs(adjustedValue - baseline.Mean)
				} else {
					continue
				}
			}
			
			threshold := ad.getThresholdForMetricType(metric.Type)
			anomalyDetected = math.Abs(deviation) > threshold
		} else {
			deviation = math.Abs(adjustedValue - baseline.Median) / baseline.MAD
			
			madThreshold := ad.getThresholdForMetricType(metric.Type) * 0.6745
			anomalyDetected = deviation > madThreshold
			
			deviation = deviation * 1.4826
		}

		if anomalyDetected {
			threshold := ad.getThresholdForMetricType(metric.Type)
			severity := ad.calculateSeverity(deviation, threshold)

			anomaly := models.Anomaly{
				MetricName:     metric.Name,
				MetricType:     metric.Type,
				Timestamp:      metric.Timestamp,
				Value:          metric.Value,
				Baseline:       baseline.Mean,
				DeviationScore: deviation,
				Severity:       severity,
				Labels:         metric.Labels,
				Unit:           metric.Unit,
			}

			anomalies = append(anomalies, anomaly)

			ad.logger.Info("Anomaly detected",
				zap.String("metric", metric.Name),
				zap.Float64("value", metric.Value),
				zap.Float64("adjusted_value", adjustedValue),
				zap.Float64("baseline", baseline.Mean),
				zap.Float64("deviation", deviation),
				zap.String("severity", severity),
				zap.String("method", string(detectionMethod)))
		}
	}

	return anomalies, nil
}

func (ad *AnomalyDetector) getOrCreateBaseline(metricName string) (*Baseline, error) {
	ad.mutex.RLock()
	baseline, exists := ad.baselines[metricName]
	ad.mutex.RUnlock()

	if exists {
		return baseline, nil
	}

	baseline = &Baseline{
		Mean:          0,
		StdDev:        0,
		Median:        0,
		MAD:           0,
		Min:           math.MaxFloat64,
		Max:           -math.MaxFloat64,
		LastUpdated:   time.Now(),
		DataPoints:    0,
		Values:        make([]float64, 0, 100), // Store recent values for adaptive baseline
		DailyPattern:  make(map[int]float64),   // Hour of day (0-23)
		WeeklyPattern: make(map[int]float64),   // Day of week (0-6)
	}

	ad.mutex.Lock()
	ad.baselines[metricName] = baseline
	ad.mutex.Unlock()

	return baseline, nil
}

func (ad *AnomalyDetector) updateBaseline(metricName string, value float64, timestamp time.Time) {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	baseline, exists := ad.baselines[metricName]
	if !exists {
		return
	}

	baseline.Values = append(baseline.Values, value)
	if len(baseline.Values) > 100 {
		baseline.Values = baseline.Values[1:]
	}

	baseline.DataPoints++

	if value < baseline.Min {
		baseline.Min = value
	}
	if value > baseline.Max {
		baseline.Max = value
	}

	if baseline.DataPoints == 1 {
		baseline.Mean = value
		baseline.StdDev = 0
		baseline.Median = value
		baseline.MAD = 0
	} else {
		adaptiveRate := ad.config.AdaptiveRate
		if adaptiveRate <= 0 {
			adaptiveRate = 0.1 // Default adaptive rate
		}

		oldMean := baseline.Mean
		baseline.Mean = oldMean + adaptiveRate*(value-oldMean)

		variance := baseline.StdDev * baseline.StdDev
		variance = (1-adaptiveRate)*variance + adaptiveRate*math.Pow(value-baseline.Mean, 2)
		baseline.StdDev = math.Sqrt(variance)

		// Update median and MAD if we have enough data points
		if len(baseline.Values) >= ad.config.MinDataPoints {
			valuesCopy := make([]float64, len(baseline.Values))
			copy(valuesCopy, baseline.Values)
			sort.Float64s(valuesCopy)
			
			n := len(valuesCopy)
			if n%2 == 0 {
				baseline.Median = (valuesCopy[n/2-1] + valuesCopy[n/2]) / 2
			} else {
				baseline.Median = valuesCopy[n/2]
			}
			
			deviations := make([]float64, n)
			for i, v := range valuesCopy {
				deviations[i] = math.Abs(v - baseline.Median)
			}
			sort.Float64s(deviations)
			
			if n%2 == 0 {
				baseline.MAD = (deviations[n/2-1] + deviations[n/2]) / 2
			} else {
				baseline.MAD = deviations[n/2]
			}
			
			if baseline.MAD < 0.0001 {
				baseline.MAD = 0.0001
			}
		}
	}

	hourOfDay := timestamp.Hour()
	dayOfWeek := int(timestamp.Weekday())
	
	adaptiveRate := ad.config.AdaptiveRate
	if adaptiveRate <= 0 {
		adaptiveRate = 0.1
	}
	
	if oldValue, exists := baseline.DailyPattern[hourOfDay]; exists {
		baseline.DailyPattern[hourOfDay] = oldValue + adaptiveRate*(value-oldValue)
	} else {
		baseline.DailyPattern[hourOfDay] = value
	}
	
	if oldValue, exists := baseline.WeeklyPattern[dayOfWeek]; exists {
		baseline.WeeklyPattern[dayOfWeek] = oldValue + adaptiveRate*(value-oldValue)
	} else {
		baseline.WeeklyPattern[dayOfWeek] = value
	}

	baseline.LastUpdated = time.Now()
}

func (ad *AnomalyDetector) getThresholdForMetricType(metricType models.MetricType) float64 {
	threshold := ad.config.AnomalyThreshold

	switch ad.config.SensitivityLevel {
	case "low":
		threshold *= 1.5
	case "high":
		threshold *= 0.7
	}

	switch metricType {
	case models.MetricTypeError:
		threshold *= 0.8 // More sensitive for errors
	case models.MetricTypeSaturation:
		threshold *= 1.2 // Less sensitive for saturation
	}

	return threshold
}

func (ad *AnomalyDetector) calculateSeverity(deviation float64, threshold float64) string {
	absDeviation := math.Abs(deviation)

	if absDeviation > threshold*2 {
		return "high"
	} else if absDeviation > threshold*1.5 {
		return "medium"
	} else {
		return "low"
	}
}

func (ad *AnomalyDetector) SetBaseline(metricName string, mean, stdDev float64) {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	baseline, exists := ad.baselines[metricName]
	if !exists {
		baseline = &Baseline{
			Values: make([]float64, 0, 100),
		}
		ad.baselines[metricName] = baseline
	}

	baseline.Mean = mean
	baseline.StdDev = stdDev
	baseline.LastUpdated = time.Now()
	baseline.DataPoints = ad.config.MinDataPoints // Ensure it has enough data points
}

func (ad *AnomalyDetector) GetBaseline(metricName string) (*Baseline, bool) {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	baseline, exists := ad.baselines[metricName]
	return baseline, exists
}

func (ad *AnomalyDetector) ResetBaseline(metricName string) {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	delete(ad.baselines, metricName)
}
