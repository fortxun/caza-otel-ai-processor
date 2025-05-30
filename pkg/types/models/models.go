package models

import (
	"time"
)

type MetricType string

const (
	MetricTypeLatency    MetricType = "latency"
	MetricTypeTraffic    MetricType = "traffic"
	MetricTypeError      MetricType = "error"
	MetricTypeSaturation MetricType = "saturation"
)

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
	AnomalyID          string    `json:"anomaly_id"`
	Description        string    `json:"description"`
	Confidence         float64   `json:"confidence"`
	RelatedMetrics     []string  `json:"related_metrics"`
	Recommendations    []string  `json:"recommendations"`
	KnowledgeBaseMatch bool      `json:"knowledge_base_match"`
	KnowledgeBaseID    string    `json:"knowledge_base_id,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}
