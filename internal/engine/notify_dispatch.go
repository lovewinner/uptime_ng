package engine

import (
	"log"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/config"
	"uptime_ng/internal/model"
	"uptime_ng/internal/notifier"
)

type NotifyDispatch struct {
	DB *gorm.DB
}

func NewNotifyDispatch(db *gorm.DB) *NotifyDispatch {
	return &NotifyDispatch{DB: db}
}

func (d *NotifyDispatch) Send(monitor *model.Monitor, heartbeat model.Heartbeat, isFirstBeat bool, prevStatus uint16) {
	isUp := heartbeat.Status == model.StatusUP

	var mnList []model.MonitorNotification
	d.DB.Where("monitor_id = ?", monitor.ID).Find(&mnList)

	notifications := make([]model.Notification, 0, len(mnList))
	for _, mn := range mnList {
		var notif model.Notification
		if err := d.DB.First(&notif, mn.NotificationID).Error; err != nil {
			continue
		}
		notifications = append(notifications, notif)
	}

	for _, notif := range activeNotifications(notifications) {
		switch notif.Type {
		case model.NotificationTypeFeishu:
			d.sendFeishu(notif, monitor, isUp, heartbeat.Msg)
		case model.NotificationTypeEmail:
			d.sendEmail(notif, monitor, isUp, heartbeat.Msg)
		default:
			log.Printf("Unknown notification type: %s", notif.Type)
		}
	}

	// Also send to global feishu webhook if configured
	if config.AppConfig != nil && config.AppConfig.Feishu.WebhookURL != "" {
		if !hasNotificationType(notifications, model.NotificationTypeFeishu) {
			notifier.SendFeishuAlert(config.AppConfig.Feishu.WebhookURL, monitor.Name, monitor.Type, isUp, heartbeat.Msg)
		}
	}
}

func (d *NotifyDispatch) sendFeishu(notif model.Notification, monitor *model.Monitor, isUp bool, msg string) {
	configMap, err := notifier.ParseNotificationConfig(notif.Config)
	if err != nil {
		log.Printf("[feishu] notification %s has invalid config: %v", notif.Name, err)
		return
	}
	webhookURL := configMap.WebhookURL()
	if webhookURL == "" {
		log.Printf("[feishu] notification %s has no webhook_url", notif.Name)
		return
	}

	notifier.SendFeishuAlert(webhookURL, monitor.Name, monitor.Type, isUp, msg)
	log.Printf("[feishu] Alert sent for %s via %s", monitor.Name, notif.Name)
}

func (d *NotifyDispatch) sendEmail(notif model.Notification, monitor *model.Monitor, isUp bool, msg string) {
	if config.AppConfig.SMTP.Host == "" {
		log.Printf("[email] SMTP not configured, skipping email notification %s", notif.Name)
		return
	}

	configMap, err := notifier.ParseNotificationConfig(notif.Config)
	if err != nil {
		log.Printf("[email] notification %s has invalid config: %v", notif.Name, err)
		return
	}
	content, ok := notifier.BuildEmailAlertContent(monitor, configMap, isUp, msg)
	if !ok {
		log.Printf("[email] notification %s has no recipient", notif.Name)
		return
	}

	n := notifier.NewEmailNotifierFromConfig(content.To)
	if err := n.Send(content.Subject, content.Body); err != nil {
		log.Printf("[email] failed to send to %s: %v", content.To, err)
	} else {
		log.Printf("[email] Alert sent for %s to %s", monitor.Name, content.To)
	}
}

func (d *NotifyDispatch) markIncident(db *gorm.DB, monitorID uint, monitorName string, prevStatus, newStatus uint16, msg string) {
	if prevStatus == model.StatusUP && (newStatus == model.StatusDown || newStatus == model.StatusPending) {
		incident := model.Incident{
			MonitorID: monitorID,
			Title:     monitorName + " went " + statusLabel(newStatus),
			Status:    model.StatusDown,
			StartedAt: time.Now(),
			Msg:       msg,
		}
		db.Create(&incident)
		log.Printf("Incident created: %s went %s", monitorName, statusLabel(newStatus))
	}

	if (prevStatus == model.StatusDown || prevStatus == model.StatusPending) && newStatus == model.StatusUP {
		var recentIncident model.Incident
		err := db.Where("monitor_id = ? AND status = ?", monitorID, model.StatusDown).
			Order("started_at DESC").First(&recentIncident).Error
		if err == nil {
			now := time.Now()
			recentIncident.EndedAt = &now
			recentIncident.DurationSec = uint32(now.Sub(recentIncident.StartedAt).Seconds())
			recentIncident.Status = model.StatusUP
			recentIncident.Title = monitorName + " recovered"
			db.Save(&recentIncident)
			log.Printf("Incident resolved: %s recovered after %ds", monitorName, recentIncident.DurationSec)
		}
	}
}

func statusLabel(status uint16) string {
	switch status {
	case model.StatusUP:
		return "UP"
	case model.StatusDown:
		return "DOWN"
	case model.StatusPending:
		return "PENDING"
	default:
		return "UNKNOWN"
	}
}
