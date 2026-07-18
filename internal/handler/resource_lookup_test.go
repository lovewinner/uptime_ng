package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestUserNotificationLooksUpOwnedNotificationOnly(t *testing.T) {
	db := testDB(t)
	owned := model.Notification{UserID: 1, Name: "owned", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	other := model.Notification{UserID: 2, Name: "other", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("create owned notification: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other notification: %v", err)
	}

	got, err := userNotification(db, 1, owned.ID)
	if err != nil {
		t.Fatalf("userNotification owned: %v", err)
	}
	if got.ID != owned.ID {
		t.Fatalf("got notification id=%d want %d", got.ID, owned.ID)
	}
	if _, err := userNotification(db, 1, other.ID); err == nil {
		t.Fatalf("expected cross-user notification lookup to fail")
	}
}

func TestUserMaintenanceWindowLooksUpOwnedWindowOnly(t *testing.T) {
	db := testDB(t)
	owned := model.MaintenanceWindow{UserID: 1, Name: "owned"}
	other := model.MaintenanceWindow{UserID: 2, Name: "other"}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("create owned maintenance window: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other maintenance window: %v", err)
	}

	got, err := userMaintenanceWindow(db, 1, owned.ID)
	if err != nil {
		t.Fatalf("userMaintenanceWindow owned: %v", err)
	}
	if got.ID != owned.ID {
		t.Fatalf("got maintenance window id=%d want %d", got.ID, owned.ID)
	}
	if _, err := userMaintenanceWindow(db, 1, other.ID); err == nil {
		t.Fatalf("expected cross-user maintenance window lookup to fail")
	}
}
