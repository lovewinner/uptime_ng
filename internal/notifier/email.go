package notifier

import (
	"fmt"
	"log"
	"net/smtp"

	"gorm.io/gorm"

	"uptime_ng/internal/config"
	"uptime_ng/internal/model"
)

type EmailNotifier struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       string
}

func NewEmailNotifierFromConfig(to string) *EmailNotifier {
	cfg := config.AppConfig.SMTP
	return &EmailNotifier{
		SMTPHost: cfg.Host,
		SMTPPort: cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		From:     cfg.From,
		To:       to,
	}
}

func NewEmailNotifierFromJSON(configJSON string, to string) *EmailNotifier {
	cfg := config.AppConfig
	return &EmailNotifier{
		SMTPHost: cfg.SMTP.Host,
		SMTPPort: cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		To:       to,
	}
}

func (n *EmailNotifier) Send(subject, body string) error {
	if n.SMTPHost == "" {
		return fmt.Errorf("SMTP not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		n.From, n.To, subject, body)

	addr := fmt.Sprintf("%s:%s", n.SMTPHost, uintToStr(n.SMTPPort))

	auth := smtp.PlainAuth("", n.Username, n.Password, n.SMTPHost)
	return smtp.SendMail(addr, auth, n.From, []string{n.To}, []byte(msg))
}

func SendEmailAlert(db *gorm.DB, monitor *model.Monitor, isUp bool, msg string) {
	if config.AppConfig.SMTP.Host == "" {
		return
	}

	var mnList []model.MonitorNotification
	db.Where("monitor_id = ?", monitor.ID).Find(&mnList)

	for _, mn := range mnList {
		var notif model.Notification
		if err := db.First(&notif, mn.NotificationID).Error; err != nil {
			continue
		}
		if notif.Type != "email" || !notif.Active {
			continue
		}

		statusText := "DOWN"
		if isUp {
			statusText = "UP (已恢复)"
		}

		subject := fmt.Sprintf("[uptime_ng] %s - %s", monitor.Name, statusText)
		htmlBody := fmt.Sprintf(`
			<h2>监控告警</h2>
			<table border="1" cellpadding="8" style="border-collapse:collapse">
				<tr><td><b>监控项</b></td><td>%s</td></tr>
				<tr><td><b>类型</b></td><td>%s</td></tr>
				<tr><td><b>状态</b></td><td style="color:%s"><b>%s</b></td></tr>
				<tr><td><b>详情</b></td><td>%s</td></tr>
			</table>
			<p style="color:#999;font-size:12px">来自 uptime_ng 监控系统</p>
		`, monitor.Name, monitor.Type, statusColor(isUp), statusText, msg)

		n := NewEmailNotifierFromConfig("dummy")
		if err := n.Send(subject, htmlBody); err != nil {
			log.Printf("[email] failed to send: %v", err)
		}
	}
}

func statusColor(isUp bool) string {
	if isUp {
		return "green"
	}
	return "red"
}

func uintToStr(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	s := ""
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func FormatEmailTemplate(m *model.Monitor, status string, msg string) string {
	return fmt.Sprintf(`<html><body>
<h2>监控告警 - %s</h2>
<p><b>监控项:</b> %s</p>
<p><b>类型:</b> %s</p>
<p><b>状态:</b> %s</p>
<p><b>时间:</b> %s</p>
<p><b>详情:</b> %s</p>
</body></html>`, m.Name, m.Name, m.Type, status, msg, msg)
}