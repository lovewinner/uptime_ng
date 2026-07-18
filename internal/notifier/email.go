package notifier

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strconv"
	"strings"

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

func (n *EmailNotifier) Send(subject, body string) error {
	if n.SMTPHost == "" {
		return fmt.Errorf("SMTP not configured")
	}
	recipients := splitRecipients(n.To)
	if len(recipients) == 0 {
		return fmt.Errorf("email recipient not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		n.From, n.To, subject, body)

	addr := fmt.Sprintf("%s:%s", n.SMTPHost, strconv.Itoa(n.SMTPPort))

	auth := smtp.PlainAuth("", n.Username, n.Password, n.SMTPHost)
	if n.SMTPPort == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: n.SMTPHost})
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, n.SMTPHost)
		if err != nil {
			return err
		}
		defer client.Quit()
		return sendWithClient(client, auth, n.From, recipients, []byte(msg))
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Quit()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: n.SMTPHost}); err != nil {
			return err
		}
	}
	return sendWithClient(client, auth, n.From, recipients, []byte(msg))
}

func sendWithClient(client *smtp.Client, auth smtp.Auth, from string, recipients []string, msg []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func splitRecipients(raw string) []string {
	raw = strings.ReplaceAll(raw, ";", ",")
	parts := strings.Split(raw, ",")
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			recipients = append(recipients, part)
		}
	}
	return recipients
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
		if notif.Type != model.NotificationTypeEmail || !notif.Active {
			continue
		}
		configMap, err := ParseNotificationConfig(notif.Config)
		if err != nil {
			log.Printf("[email] notification %s has invalid config: %v", notif.Name, err)
			continue
		}
		content, ok := BuildEmailAlertContent(monitor, configMap, isUp, msg)
		if !ok {
			continue
		}

		n := NewEmailNotifierFromConfig(content.To)
		if err := n.Send(content.Subject, content.Body); err != nil {
			log.Printf("[email] failed to send: %v", err)
		}
	}
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
