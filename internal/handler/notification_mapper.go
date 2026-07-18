package handler

import (
	"fmt"

	"uptime_ng/internal/model"
	"uptime_ng/internal/notifier"
)

type notificationTestTarget struct {
	message      string
	webhookURL   string
	emailTo      string
	emailSubject string
	emailBody    string
}

func notificationFromRequest(req CreateNotificationRequest, userID uint) (model.Notification, *requestValidationError) {
	if err := validateNotificationType(req.Type); err != nil {
		return model.Notification{}, err
	}
	return model.Notification{
		UserID: userID,
		Name:   req.Name,
		Type:   req.Type,
		Config: req.Config,
		Active: true,
	}, nil
}

func applyNotificationRequest(notif *model.Notification, req CreateNotificationRequest) *requestValidationError {
	if err := validateNotificationType(req.Type); err != nil {
		return err
	}
	notif.Name = req.Name
	notif.Type = req.Type
	notif.Config = req.Config
	return nil
}

func validateNotificationType(notificationType string) *requestValidationError {
	if notificationType == model.NotificationTypeFeishu || notificationType == model.NotificationTypeEmail {
		return nil
	}
	return &requestValidationError{code: "invalid_notification_type", message: "type must be feishu or email"}
}

func notificationTestTargetFromConfig(notif model.Notification, cfg notifier.NotificationConfig) (notificationTestTarget, *requestValidationError) {
	msg := fmt.Sprintf("来自 uptime_ng 的测试消息。通知: %s (%s)", notif.Name, notif.Type)
	target := notificationTestTarget{message: msg}

	switch notif.Type {
	case model.NotificationTypeFeishu:
		target.webhookURL = cfg.WebhookURL()
		if target.webhookURL == "" {
			return target, &requestValidationError{code: "missing_webhook_url", message: "missing webhook_url"}
		}
	case model.NotificationTypeEmail:
		target.emailTo = cfg.EmailRecipients()
		if target.emailTo == "" {
			return target, &requestValidationError{code: "missing_email_recipient", message: "missing email recipient"}
		}
		target.emailSubject = "[uptime_ng] 通知测试"
		target.emailBody = "<p>" + msg + "</p>"
	default:
		return target, &requestValidationError{code: "unsupported_notification_type", message: "unsupported notification type"}
	}

	return target, nil
}
