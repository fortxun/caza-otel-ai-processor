package models

import (
	"time"
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

type Anomaly struct {
	MetricName     string            `json:"metric_name"`
	MetricType     MetricType        `json:"metric_type"`
	Timestamp      time.Time         `json:"timestamp"`
	Value          float64           `json:"value"`
	Baseline       float64           `json:"baseline"`
	DeviationScore float64           `json:"deviation_score"`
	Severity       string            `json:"severity"`
	Labels         map[string]string `json:"labels"`
	Unit           string            `json:"unit"`
}

type RootCause struct {
	Description     string   `json:"description"`
	Confidence      float64  `json:"confidence"`
	RelatedMetrics  []string `json:"related_metrics"`
	Recommendations []string `json:"recommendations"`
	LLMGenerated    bool     `json:"llm_generated"`
	Reasoning       string   `json:"reasoning"`
}

type Alert struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Anomalies []Anomaly `json:"anomalies"`
}
