package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestRefreshMonitorAssociationsHonorsIndependentFlags(t *testing.T) {
	db := testDB(t)
	tx := db.Begin()

	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	if err := tx.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	oldNotif := model.Notification{UserID: 1, Name: "old", Type: model.NotificationTypeEmail, Config: `{"to":"old@example.com"}`, Active: true}
	newNotif := model.Notification{UserID: 1, Name: "new", Type: model.NotificationTypeEmail, Config: `{"to":"new@example.com"}`, Active: true}
	if err := tx.Create(&oldNotif).Error; err != nil {
		t.Fatalf("create old notif: %v", err)
	}
	if err := tx.Create(&newNotif).Error; err != nil {
		t.Fatalf("create new notif: %v", err)
	}

	attachMonitorAssociations(tx, monitor.ID, []uint{oldNotif.ID}, []string{"prod"}, []string{"#123456"})
	refreshMonitorAssociations(tx, monitor.ID, []uint{newNotif.ID}, nil, nil, true, false)
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit: %v", err)
	}

	var links []model.MonitorNotification
	db.Where("monitor_id = ?", monitor.ID).Find(&links)
	if len(links) != 1 || links[0].NotificationID != newNotif.ID {
		t.Fatalf("notification links=%+v want new=%d", links, newNotif.ID)
	}

	var tags []model.MonitorTag
	db.Where("monitor_id = ?", monitor.ID).Find(&tags)
	if len(tags) != 1 {
		t.Fatalf("tags were refreshed unexpectedly: %+v", tags)
	}
}
