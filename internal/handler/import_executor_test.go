package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestImportNotificationsKeepsMaskedExistingSecret(t *testing.T) {
	db := testDB(t)
	userID := uint(1)
	existing := model.Notification{
		UserID: userID,
		Name:   "ops",
		Type:   "feishu",
		Config: `{"webhook_url":"https://real"}`,
		Active: true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing: %v", err)
	}

	tx := db.Begin()
	importNotifications(tx, userID, []ExportNotification{{
		Name:   "ops",
		Type:   "email",
		Config: `{"webhook_url":"***"}`,
	}}, "overwrite")
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit: %v", err)
	}

	var got model.Notification
	if err := db.First(&got, existing.ID).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if got.Type != "email" {
		t.Fatalf("type=%s want email", got.Type)
	}
	if got.Config != existing.Config {
		t.Fatalf("masked config overwrote secret: %s", got.Config)
	}
}

func TestSyncImportedMonitorSchedulers(t *testing.T) {
	scheduler := &fakeScheduler{}
	syncImportedMonitorSchedulers(scheduler, []model.Monitor{
		{ID: 1, Type: model.MonitorTypeGroup, Active: true},
		{ID: 2, Type: model.MonitorTypeHTTP, Active: true},
		{ID: 3, Type: model.MonitorTypeTCP, Active: false},
	})

	if len(scheduler.calls) != 3 {
		t.Fatalf("calls=%+v", scheduler.calls)
	}
	want := []schedulerCall{
		{action: "stop", id: 1},
		{action: "restart", id: 2},
		{action: "stop", id: 3},
	}
	for i := range want {
		if scheduler.calls[i].action != want[i].action || scheduler.calls[i].id != want[i].id {
			t.Fatalf("call[%d]=%+v want %+v", i, scheduler.calls[i], want[i])
		}
	}
}
