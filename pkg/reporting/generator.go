package reporting

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fortxun/idop/pkg/types/models"
	"go.uber.org/zap"
)

type ReportGenerator struct {
	logger *zap.Logger
}

type Report struct {
	ID               string           `json:"id"`
	Timestamp        time.Time        `json:"timestamp"`
	Anomalies        []models.Anomaly `json:"anomalies"`
	RootCauses       []models.RootCause `json:"root_causes"`
	Summary          string           `json:"summary"`
	Recommendations  []string         `json:"recommendations"`
	OverallSeverity  string           `json:"overall_severity"`
}

func NewReportGenerator(logger *zap.Logger) *ReportGenerator {
	return &ReportGenerator{
		logger: logger,
	}
}

func (rg *ReportGenerator) GenerateReport(anomalies []models.Anomaly, rootCauses []models.RootCause) *Report {
	reportID := fmt.Sprintf("IDOP-REPORT-%d", time.Now().Unix())

	overallSeverity := "low"
	for _, anomaly := range anomalies {
		if anomaly.Severity == "high" {
			overallSeverity = "high"
			break
		} else if anomaly.Severity == "medium" && overallSeverity != "high" {
			overallSeverity = "medium"
		}
	}

	recommendations := make([]string, 0)
	recommendationSet := make(map[string]bool)

	for _, rootCause := range rootCauses {
		for _, rec := range rootCause.Recommendations {
			if !recommendationSet[rec] {
				recommendations = append(recommendations, rec)
				recommendationSet[rec] = true
			}
		}
	}

	if len(rootCauses) == 0 {
		recommendations = rg.generateGenericRecommendations(anomalies)
	}

	summary := rg.generateSummary(anomalies, rootCauses, overallSeverity)

	report := &Report{
		ID:               reportID,
		Timestamp:        time.Now(),
		Anomalies:        anomalies,
		RootCauses:       rootCauses,
		Summary:          summary,
		Recommendations:  recommendations,
		OverallSeverity:  overallSeverity,
	}

	rg.logger.Info("Generated report",
		zap.String("report_id", reportID),
		zap.String("severity", overallSeverity),
		zap.Int("anomalies", len(anomalies)),
		zap.Int("root_causes", len(rootCauses)))

	return report
}

func (rg *ReportGenerator) generateSummary(anomalies []models.Anomaly, rootCauses []models.RootCause, severity string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Database Observability Report (%s severity)\n\n", severity))

	sb.WriteString(fmt.Sprintf("Detected %d anomalies across database metrics:\n", len(anomalies)))
	
	latencyAnomalies := 0
	trafficAnomalies := 0
	errorAnomalies := 0
	saturationAnomalies := 0
	
	for _, anomaly := range anomalies {
		switch anomaly.MetricType {
		case models.MetricTypeLatency:
			latencyAnomalies++
		case models.MetricTypeTraffic:
			trafficAnomalies++
		case models.MetricTypeError:
			errorAnomalies++
		case models.MetricTypeSaturation:
			saturationAnomalies++
		}
	}
	
	if latencyAnomalies > 0 {
		sb.WriteString(fmt.Sprintf("- %d latency anomalies\n", latencyAnomalies))
	}
	if trafficAnomalies > 0 {
		sb.WriteString(fmt.Sprintf("- %d traffic anomalies\n", trafficAnomalies))
	}
	if errorAnomalies > 0 {
		sb.WriteString(fmt.Sprintf("- %d error anomalies\n", errorAnomalies))
	}
	if saturationAnomalies > 0 {
		sb.WriteString(fmt.Sprintf("- %d saturation anomalies\n", saturationAnomalies))
	}
	
	sb.WriteString("\n")

	if len(rootCauses) > 0 {
		sb.WriteString("Root causes identified:\n")
		for i, rootCause := range rootCauses {
			if i >= 3 {
				sb.WriteString(fmt.Sprintf("- ... and %d more root causes\n", len(rootCauses)-3))
				break
			}
			sb.WriteString(fmt.Sprintf("- %s\n", rootCause.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (rg *ReportGenerator) generateGenericRecommendations(anomalies []models.Anomaly) []string {
	recommendations := make([]string, 0)
	recommendationSet := make(map[string]bool)

	for _, anomaly := range anomalies {
		var rec string
		switch anomaly.MetricType {
		case models.MetricTypeLatency:
			rec = "Investigate query performance and consider optimizing slow queries."
		case models.MetricTypeTraffic:
			rec = "Check for unusual traffic patterns and consider scaling resources if needed."
		case models.MetricTypeError:
			rec = "Review error logs and fix any recurring errors."
		case models.MetricTypeSaturation:
			rec = "Monitor resource utilization and consider scaling up if consistently high."
		}

		if rec != "" && !recommendationSet[rec] {
			recommendations = append(recommendations, rec)
			recommendationSet[rec] = true
		}
	}

	generalRecs := []string{
		"Review recent application deployments or changes that might have affected database performance.",
		"Check for scheduled maintenance or background jobs that might be impacting performance.",
		"Verify that database indexes are properly configured and up-to-date.",
	}

	for _, rec := range generalRecs {
		if !recommendationSet[rec] {
			recommendations = append(recommendations, rec)
			recommendationSet[rec] = true
		}
	}

	return recommendations
}

func (rg *ReportGenerator) FormatJSON(report *Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %w", err)
	}
	
	return string(data), nil
}

func (rg *ReportGenerator) FormatText(report *Report) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Database Observability Report\n"))
	sb.WriteString(fmt.Sprintf("ID: %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Severity: %s\n\n", report.OverallSeverity))

	sb.WriteString(report.Summary)

	sb.WriteString("Anomalies:\n")
	for i, anomaly := range report.Anomalies {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... and %d more anomalies\n", len(report.Anomalies)-10))
			break
		}
		
		sb.WriteString(fmt.Sprintf("- %s: %.2f %s (%.2f std dev, %s severity)\n",
			anomaly.MetricName,
			anomaly.Value,
			anomaly.Unit,
			anomaly.DeviationScore,
			anomaly.Severity))
	}
	sb.WriteString("\n")

	if len(report.RootCauses) > 0 {
		sb.WriteString("Root Causes:\n")
		for _, rootCause := range report.RootCauses {
			sb.WriteString(fmt.Sprintf("- %s\n", rootCause.Description))
			sb.WriteString(fmt.Sprintf("  Confidence: %.2f\n", rootCause.Confidence))
			
			if len(rootCause.RelatedMetrics) > 0 {
				sb.WriteString("  Related Metrics:\n")
				for _, metric := range rootCause.RelatedMetrics {
					sb.WriteString(fmt.Sprintf("  - %s\n", metric))
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Recommendations:\n")
	for i, rec := range report.Recommendations {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
	}

	return sb.String()
}
