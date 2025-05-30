package llm

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/fortxun/caza-otel-ai-processor/pkg/analysis"
	"github.com/fortxun/caza-otel-ai-processor/pkg/metrics"
	"github.com/fortxun/caza-otel-ai-processor/pkg/types/models"
)

type PromptTemplates struct {
	RootCauseAnalysis    string
	AnomalyDescription   string
	RecommendationEngine string
	ConfidenceScoring    string
	SystemContext        string
}

func DefaultPromptTemplates() PromptTemplates {
	return PromptTemplates{
		SystemContext: `You are an expert database administrator and performance analyst. 
Your task is to analyze database metrics, identify anomalies, determine root causes, 
and provide actionable recommendations. Focus on being precise, technical, and concise.
Base your analysis on the provided metrics, correlations, and knowledge base information.
Provide confidence scores for your analysis based on the strength of evidence.`,

		RootCauseAnalysis: `
# Database Anomaly Analysis Task

## Anomaly Details
- Metric Name: {{.Anomaly.MetricName}}
- Metric Type: {{.Anomaly.MetricType}}
- Current Value: {{printf "%.2f" .Anomaly.Value}} {{.Anomaly.Unit}}
- Baseline Value: {{printf "%.2f" .Anomaly.Baseline}} {{.Anomaly.Unit}}
- Deviation Score: {{printf "%.2f" .Anomaly.DeviationScore}}
- Severity: {{.Anomaly.Severity}}
- Timestamp: {{.Anomaly.Timestamp.Format "2006-01-02 15:04:05"}}

## System Context
{{if .Labels}}
### Labels
{{range $key, $value := .Anomaly.Labels}}
- {{$key}}: {{$value}}
{{end}}
{{end}}

## Correlated Metrics
{{if .Correlations}}
The following metrics show significant correlation with the anomalous metric:
{{range .Correlations}}
- {{.TargetMetric}} ({{printf "%.2f" .Coefficient}} correlation, method: {{.Method}}{{if ne .TimeOffset 0}}, lag: {{.TimeOffset}}{{end}})
{{end}}
{{else}}
No significant metric correlations were found.
{{end}}

## Knowledge Base Matches
{{if .KnowledgeMatches}}
The following knowledge base entries match this anomaly pattern:
{{range .KnowledgeMatches}}
- {{.Name}}: {{.Description}}
  - Typical patterns: {{.MetricPattern}}
  - Typical recommendations: {{range .Recommendations}}{{.}}, {{end}}
{{end}}
{{else}}
No knowledge base entries directly match this anomaly pattern.
{{end}}

## Historical Context
{{if .HistoricalAnomalies}}
Similar anomalies have occurred:
{{range .HistoricalAnomalies}}
- {{.Timestamp.Format "2006-01-02 15:04:05"}}: {{.MetricName}} ({{printf "%.2f" .DeviationScore}} deviation)
{{end}}
{{else}}
No similar historical anomalies were found.
{{end}}

## Analysis Task
Based on the information above:

1. Identify the most likely root cause of this anomaly
2. Explain the reasoning behind your conclusion
3. Provide a confidence score (0-100%) for your analysis
4. List 2-3 specific, actionable recommendations to address the issue
5. Explain any potential impact if the issue is not addressed

Format your response as a JSON object with the following structure:
{
  "root_cause": "Brief description of the root cause",
  "explanation": "Detailed explanation of your reasoning",
  "confidence": 85,
  "recommendations": [
    "First specific recommendation",
    "Second specific recommendation",
    "Third specific recommendation (if applicable)"
  ],
  "impact": "Description of potential impact if not addressed"
}
`,

		AnomalyDescription: `
# Database Metric Anomaly Description Task

## Anomaly Details
- Metric Name: {{.Anomaly.MetricName}}
- Metric Type: {{.Anomaly.MetricType}}
- Current Value: {{printf "%.2f" .Anomaly.Value}} {{.Anomaly.Unit}}
- Baseline Value: {{printf "%.2f" .Anomaly.Baseline}} {{.Anomaly.Unit}}
- Deviation Score: {{printf "%.2f" .Anomaly.DeviationScore}}
- Severity: {{.Anomaly.Severity}}
- Timestamp: {{.Anomaly.Timestamp.Format "2006-01-02 15:04:05"}}

## Task
Provide a concise, technical description of this anomaly in 1-2 sentences.
Focus on what the anomaly represents in database performance terms.

Format your response as a plain text string without any JSON formatting or additional context.
`,

		RecommendationEngine: `
# Database Performance Recommendation Task

## Anomaly Context
{{range .Anomalies}}
- {{.MetricName}} ({{.MetricType}}): {{printf "%.2f" .Value}} {{.Unit}} (baseline: {{printf "%.2f" .Baseline}} {{.Unit}}, deviation: {{printf "%.2f" .DeviationScore}})
{{end}}

## Root Cause Analysis
{{if .RootCauses}}
Identified root causes:
{{range .RootCauses}}
- {{.Description}} (confidence: {{printf "%.0f%%" (mul .Confidence 100)}})
{{end}}
{{else}}
No specific root causes have been identified.
{{end}}

## Knowledge Base Context
{{if .KnowledgeMatches}}
Relevant knowledge base entries:
{{range .KnowledgeMatches}}
- {{.Name}}: {{.Description}}
{{end}}
{{else}}
No specific knowledge base entries match this scenario.
{{end}}

## Task
Based on the information above, provide 3-5 specific, actionable recommendations to:
1. Address the immediate performance issues
2. Prevent similar issues in the future
3. Improve overall database performance

Format your response as a JSON array of recommendation strings:
[
  "First specific recommendation",
  "Second specific recommendation",
  "Third specific recommendation",
  "Fourth specific recommendation (if applicable)",
  "Fifth specific recommendation (if applicable)"
]
`,

		ConfidenceScoring: `
# Confidence Scoring Task

## Analysis Context
- Metric: {{.Anomaly.MetricName}} ({{.Anomaly.MetricType}})
- Deviation: {{printf "%.2f" .Anomaly.DeviationScore}}
- Root Cause: {{.RootCause.Description}}

## Evidence Strength
{{if .Correlations}}
- Correlation evidence: {{len .Correlations}} correlated metrics
- Strongest correlation: {{printf "%.2f" .StrongestCorrelation}}
{{else}}
- No correlation evidence available
{{end}}

{{if .KnowledgeMatches}}
- Knowledge base matches: {{len .KnowledgeMatches}}
{{else}}
- No knowledge base matches
{{end}}

{{if .HistoricalAnomalies}}
- Similar historical anomalies: {{len .HistoricalAnomalies}}
{{else}}
- No similar historical anomalies
{{end}}

## Task
Based on the available evidence, provide a confidence score (0-100%) for the identified root cause.
Consider the strength of correlations, knowledge base matches, and historical patterns.

Format your response as a single integer number between 0 and 100, without any additional text.
`,
	}
}

type PromptData struct {
	Anomaly             models.Anomaly
	Anomalies           []models.Anomaly
	Correlations        []analysis.CorrelationResult
	KnowledgeMatches    []analysis.KnowledgeEntry
	HistoricalAnomalies []models.Anomaly
	RootCause           models.RootCause
	RootCauses          []models.RootCause
	StrongestCorrelation float64
	MetricsData         map[string]*metrics.TimeSeriesMetric
}

func GenerateRootCauseAnalysisPrompt(
	anomaly models.Anomaly,
	correlations []analysis.CorrelationResult,
	knowledgeMatches []analysis.KnowledgeEntry,
	historicalAnomalies []models.Anomaly,
) (string, error) {
	templates := DefaultPromptTemplates()
	
	data := PromptData{
		Anomaly:             anomaly,
		Correlations:        correlations,
		KnowledgeMatches:    knowledgeMatches,
		HistoricalAnomalies: historicalAnomalies,
	}
	
	strongestCorr := 0.0
	for _, corr := range correlations {
		if math.Abs(corr.Coefficient) > math.Abs(strongestCorr) {
			strongestCorr = corr.Coefficient
		}
	}
	data.StrongestCorrelation = strongestCorr
	
	return renderTemplate(templates.SystemContext+templates.RootCauseAnalysis, data)
}

func GenerateAnomalyDescriptionPrompt(anomaly models.Anomaly) (string, error) {
	templates := DefaultPromptTemplates()
	
	data := PromptData{
		Anomaly: anomaly,
	}
	
	return renderTemplate(templates.SystemContext+templates.AnomalyDescription, data)
}

func GenerateRecommendationPrompt(
	anomalies []models.Anomaly,
	rootCauses []models.RootCause,
	knowledgeMatches []analysis.KnowledgeEntry,
) (string, error) {
	templates := DefaultPromptTemplates()
	
	data := PromptData{
		Anomalies:        anomalies,
		RootCauses:       rootCauses,
		KnowledgeMatches: knowledgeMatches,
	}
	
	return renderTemplate(templates.SystemContext+templates.RecommendationEngine, data)
}

func GenerateConfidenceScoringPrompt(
	anomaly models.Anomaly,
	rootCause models.RootCause,
	correlations []analysis.CorrelationResult,
	knowledgeMatches []analysis.KnowledgeEntry,
	historicalAnomalies []models.Anomaly,
) (string, error) {
	templates := DefaultPromptTemplates()
	
	strongestCorr := 0.0
	for _, corr := range correlations {
		if math.Abs(corr.Coefficient) > math.Abs(strongestCorr) {
			strongestCorr = corr.Coefficient
		}
	}
	
	data := PromptData{
		Anomaly:             anomaly,
		RootCause:           rootCause,
		Correlations:        correlations,
		KnowledgeMatches:    knowledgeMatches,
		HistoricalAnomalies: historicalAnomalies,
		StrongestCorrelation: strongestCorr,
	}
	
	return renderTemplate(templates.SystemContext+templates.ConfidenceScoringPrompt, data)
}

func renderTemplate(templateText string, data interface{}) (string, error) {
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
	}
	
	tmpl, err := template.New("prompt").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return strings.TrimSpace(buf.String()), nil
}
