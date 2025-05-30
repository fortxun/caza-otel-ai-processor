package llm

import (
	"strings"
)

type PromptTemplates struct {
	RCAPrompt             string
	SummaryPrompt         string
	RecommendationsPrompt string
}

func DefaultPromptTemplates() PromptTemplates {
	return PromptTemplates{
		RCAPrompt:             defaultRCAPrompt,
		SummaryPrompt:         defaultSummaryPrompt,
		RecommendationsPrompt: defaultRecommendationsPrompt,
	}
}

const defaultRCAPrompt = `You are an expert database performance analyst. Analyze the following database metrics anomalies and identify the most likely root causes.

ANOMALIES:
{{ANOMALIES}}

RELATED METRICS:
{{RELATED_METRICS}}

SYSTEM INFORMATION:
{{SYSTEM_INFO}}

Based on the above information, identify the most likely root causes for these anomalies. For each root cause:
1. Provide a clear description of the issue
2. Assign a confidence level (0.0-1.0)
3. List the related metrics that support this conclusion
4. Provide specific recommendations to address the issue

Format your response as a JSON array of root causes, each with the following structure:
[
  {
    "description": "Clear description of the root cause",
    "confidence": 0.85,
    "related_metrics": ["metric_name_1", "metric_name_2"],
    "recommendations": ["Specific action to take", "Another specific action"]
  }
]
`

const defaultSummaryPrompt = `You are an expert database performance analyst. Create a concise summary of the following database performance anomalies and their root causes.

ANOMALIES:
{{ANOMALIES}}

ROOT CAUSES:
{{ROOT_CAUSES}}

Create a clear, concise summary that:
1. Highlights the most critical issues
2. Explains the relationships between anomalies and root causes
3. Prioritizes issues based on severity and impact
4. Uses technical but accessible language

Your summary should be suitable for both technical and non-technical stakeholders.
`

const defaultRecommendationsPrompt = `You are an expert database performance analyst. Based on the following database performance anomalies and their root causes, provide specific, actionable recommendations.

ANOMALIES:
{{ANOMALIES}}

ROOT CAUSES:
{{ROOT_CAUSES}}

Provide a prioritized list of specific, actionable recommendations that:
1. Address the identified root causes
2. Can be implemented by database administrators or developers
3. Include both immediate actions and longer-term solutions
4. Consider potential trade-offs and dependencies

Format your response as a JSON array of strings, each containing a specific recommendation:
[
  "Specific recommendation 1 with clear action steps",
  "Specific recommendation 2 with clear action steps"
]
`
