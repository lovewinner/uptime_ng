package notifier

import (
	"fmt"

	"uptime_ng/internal/model"
)

type EmailAlertContent struct {
	To      string
	Subject string
	Body    string
}

func BuildEmailAlertContent(monitor *model.Monitor, cfg NotificationConfig, isUp bool, msg string) (EmailAlertContent, bool) {
	to := cfg.EmailRecipients()
	if to == "" {
		return EmailAlertContent{}, false
	}

	statusText := "DOWN"
	if isUp {
		statusText = "UP (已恢复)"
	}
	vars := map[string]string{
		"NAME":   monitor.Name,
		"TYPE":   monitor.Type,
		"STATUS": statusText,
		"MSG":    msg,
	}

	subject := cfg.Template("subject_template", vars)
	if subject == "" {
		subject = fmt.Sprintf("[uptime_ng] %s - %s", monitor.Name, statusText)
	}
	body := cfg.Template("body_template", vars)
	if body == "" {
		body = FormatEmailTemplate(monitor, statusText, msg)
	}

	return EmailAlertContent{
		To:      to,
		Subject: subject,
		Body:    body,
	}, true
}
