package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/fortxun/idop/pkg/reporting"
	"github.com/fortxun/idop/pkg/types/config"
	"go.uber.org/zap"
)

type Notifier struct {
	config *config.AlertingConfig
	logger *zap.Logger
}

func NewNotifier(config *config.AlertingConfig, logger *zap.Logger) *Notifier {
	return &Notifier{
		config: config,
		logger: logger,
	}
}

func (n *Notifier) SendAlert(ctx context.Context, report *reporting.Report) error {
	if !n.config.Enabled {
		n.logger.Debug("Alerting is disabled, skipping alert")
		return nil
	}

	n.logger.Info("Sending alert",
		zap.String("report_id", report.ID),
		zap.String("severity", report.OverallSeverity))

	if report.OverallSeverity == "low" {
		n.logger.Debug("Skipping alert for low severity report")
		return nil
	}

	var errors []error

	if n.config.Slack.WebhookURL != "" {
		if err := n.SendSlackAlert(ctx, report); err != nil {
			n.logger.Error("Failed to send Slack alert",
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if n.config.Email.SMTPHost != "" && len(n.config.Email.To) > 0 {
		if err := n.SendEmailAlert(ctx, report); err != nil {
			n.logger.Error("Failed to send email alert",
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send %d alerts", len(errors))
	}

	return nil
}

func (n *Notifier) SendSlackAlert(ctx context.Context, report *reporting.Report) error {
	n.logger.Debug("Sending Slack alert",
		zap.String("webhook_url", maskURL(n.config.Slack.WebhookURL)),
		zap.String("channel", n.config.Slack.Channel))

	var color string
	switch report.OverallSeverity {
	case "high":
		color = "#FF0000" // Red
	case "medium":
		color = "#FFA500" // Orange
	default:
		color = "#FFFF00" // Yellow
	}

	anomalyText := ""
	for i, anomaly := range report.Anomalies {
		if i >= 5 {
			anomalyText += fmt.Sprintf("... and %d more anomalies\n", len(report.Anomalies)-5)
			break
		}
		anomalyText += fmt.Sprintf("• %s: %.2f %s (%.2f std dev, %s severity)\n",
			anomaly.MetricName,
			anomaly.Value,
			anomaly.Unit,
			anomaly.DeviationScore,
			anomaly.Severity)
	}

	recommendationsText := ""
	for i, rec := range report.Recommendations {
		recommendationsText += fmt.Sprintf("%d. %s\n", i+1, rec)
	}

	payload := map[string]interface{}{
		"channel": n.config.Slack.Channel,
		"username": getSlackUsername(n.config),
		"attachments": []map[string]interface{}{
			{
				"fallback":    fmt.Sprintf("IDOP Alert: %s severity database anomalies detected", report.OverallSeverity),
				"color":       color,
				"title":       fmt.Sprintf("IDOP Alert: %s severity database anomalies detected", report.OverallSeverity),
				"title_link":  "", // Could link to a dashboard in the future
				"text":        report.Summary,
				"fields": []map[string]interface{}{
					{
						"title": "Anomalies",
						"value": anomalyText,
						"short": false,
					},
					{
						"title": "Recommendations",
						"value": recommendationsText,
						"short": false,
					},
				},
				"footer":      fmt.Sprintf("IDOP Report ID: %s", report.ID),
				"ts":          report.Timestamp.Unix(),
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.config.Slack.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned non-200 status: %d %s", resp.StatusCode, resp.Status)
	}

	n.logger.Info("Slack alert sent successfully",
		zap.String("report_id", report.ID))

	return nil
}

func (n *Notifier) SendEmailAlert(ctx context.Context, report *reporting.Report) error {
	n.logger.Debug("Sending email alert",
		zap.String("smtp_host", n.config.Email.SMTPHost),
		zap.Int("smtp_port", n.config.Email.SMTPPort),
		zap.Strings("recipients", n.config.Email.To))

	subject := fmt.Sprintf("IDOP Alert: %s severity database anomalies detected", report.OverallSeverity)

	body := fmt.Sprintf("Database Observability Report\n")
	body += fmt.Sprintf("ID: %s\n", report.ID)
	body += fmt.Sprintf("Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	body += fmt.Sprintf("Severity: %s\n\n", report.OverallSeverity)
	body += report.Summary + "\n"

	body += "Anomalies:\n"
	for i, anomaly := range report.Anomalies {
		if i >= 10 {
			body += fmt.Sprintf("... and %d more anomalies\n", len(report.Anomalies)-10)
			break
		}
		
		body += fmt.Sprintf("- %s: %.2f %s (%.2f std dev, %s severity)\n",
			anomaly.MetricName,
			anomaly.Value,
			anomaly.Unit,
			anomaly.DeviationScore,
			anomaly.Severity)
	}
	body += "\n"

	if len(report.RootCauses) > 0 {
		body += "Root Causes:\n"
		for _, rootCause := range report.RootCauses {
			body += fmt.Sprintf("- %s\n", rootCause.Description)
			body += fmt.Sprintf("  Confidence: %.2f\n", rootCause.Confidence)
		}
		body += "\n"
	}

	body += "Recommendations:\n"
	for i, rec := range report.Recommendations {
		body += fmt.Sprintf("%d. %s\n", i+1, rec)
	}

	to := strings.Join(n.config.Email.To, ", ")
	message := fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", to, n.config.Email.From, subject, body)

	auth := smtp.PlainAuth("", n.config.Email.Username, n.config.Email.Password, n.config.Email.SMTPHost)
	
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", n.config.Email.SMTPHost, n.config.Email.SMTPPort),
		auth,
		n.config.Email.From,
		n.config.Email.To,
		[]byte(message),
	)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	n.logger.Info("Email alert sent successfully",
		zap.String("report_id", report.ID),
		zap.Strings("recipients", n.config.Email.To))

	return nil
}


func maskURL(url string) string {
	if len(url) <= 8 {
		return "***"
	}
	return url[:8] + "***"
}

func getSlackUsername(config *config.AlertingConfig) string {
	if config.Slack.Username != "" {
		return config.Slack.Username
	}
	return "IDOP Alert Bot"
}
