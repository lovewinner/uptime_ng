package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestActiveNotifications(t *testing.T) {
	notifications := []model.Notification{
		{ID: 1, Type: model.NotificationTypeFeishu, Active: true},
		{ID: 2, Type: model.NotificationTypeEmail, Active: false},
		{ID: 3, Type: model.NotificationTypeEmail, Active: true},
	}

	active := activeNotifications(notifications)
	if len(active) != 2 || active[0].ID != 1 || active[1].ID != 3 {
		t.Fatalf("active=%+v", active)
	}
}

func TestHasNotificationType(t *testing.T) {
	notifications := []model.Notification{
		{ID: 1, Type: model.NotificationTypeEmail, Active: true},
		{ID: 2, Type: model.NotificationTypeFeishu, Active: false},
	}
	if !hasNotificationType(notifications, model.NotificationTypeFeishu) {
		t.Fatalf("expected feishu to be present")
	}
	if hasNotificationType(notifications, "webhook") {
		t.Fatalf("webhook should not be present")
	}
}
