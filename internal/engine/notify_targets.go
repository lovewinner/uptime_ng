package engine

import "uptime_ng/internal/model"

func hasNotificationType(notifications []model.Notification, notificationType string) bool {
	for _, notification := range notifications {
		if notification.Type == notificationType {
			return true
		}
	}
	return false
}

func activeNotifications(notifications []model.Notification) []model.Notification {
	active := make([]model.Notification, 0, len(notifications))
	for _, notification := range notifications {
		if notification.Active {
			active = append(active, notification)
		}
	}
	return active
}
