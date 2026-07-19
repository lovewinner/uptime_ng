package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestMonitorResponseIncludesAssociations(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	tag := model.Tag{Name: "prod", Color: "#123456"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	notif := model.Notification{UserID: 1, Name: "ops", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	if err := db.Create(&notif).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	db.Create(&model.MonitorTag{MonitorID: monitor.ID, TagID: tag.ID, Value: tag.Name})
	db.Create(&model.MonitorNotification{MonitorID: monitor.ID, NotificationID: notif.ID})

	resp, err := monitorResponse(db, monitor)
	if err != nil {
		t.Fatalf("monitor response: %v", err)
	}

	tags, ok := resp["tags"].([]model.Tag)
	if !ok || len(tags) != 1 || tags[0].Name != "prod" {
		t.Fatalf("tags=%+v", resp["tags"])
	}
	ids, ok := resp["notification_ids"].([]uint)
	if !ok || len(ids) != 1 || ids[0] != notif.ID {
		t.Fatalf("notification_ids=%+v", resp["notification_ids"])
	}
}

func TestMonitorResponseIgnoresCrossUserNotificationLinks(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	owned := model.Notification{UserID: 1, Name: "owned", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	other := model.Notification{UserID: 2, Name: "other", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("create owned notification: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other notification: %v", err)
	}
	db.Create(&model.MonitorNotification{MonitorID: monitor.ID, NotificationID: owned.ID})
	db.Create(&model.MonitorNotification{MonitorID: monitor.ID, NotificationID: other.ID})

	resp, err := monitorResponse(db, monitor)
	if err != nil {
		t.Fatalf("monitor response: %v", err)
	}
	ids, ok := resp["notification_ids"].([]uint)
	if !ok || len(ids) != 1 || ids[0] != owned.ID {
		t.Fatalf("notification_ids=%+v want only owned=%d", resp["notification_ids"], owned.ID)
	}
}

func TestMonitorResponseReturnsAssociationErrors(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Migrator().DropTable(&model.MonitorTag{}); err != nil {
		t.Fatalf("drop monitor_tags: %v", err)
	}

	if _, err := monitorResponse(db, monitor); err == nil {
		t.Fatal("expected monitor response error")
	}
}
