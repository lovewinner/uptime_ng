package engine

import (
	"errors"
	"log"

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

func (d *NotifyDispatch) Send(monitor *model.Monitor, heartbeat model.Heartbeat, isFirstBeat bool, prevStatus uint16) error {
	isUp := heartbeat.Status == model.StatusUP

	notifications, err := linkedNotificationsForMonitor(d.DB, monitor.UserID, monitor.ID)
	if err != nil {
		return err
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
	return nil
}

func linkedNotificationsForMonitor(db *gorm.DB, userID uint, monitorID uint) ([]model.Notification, error) {
	var notifications []model.Notification
	err := db.Model(&model.Notification{}).
		Select("notifications.*").
		Joins("JOIN monitor_notifications mn ON mn.notification_id = notifications.id").
		Where("notifications.user_id = ? AND mn.monitor_id = ?", userID, monitorID).
		Find(&notifications).Error
	return notifications, err
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
