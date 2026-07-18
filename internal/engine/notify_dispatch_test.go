package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestNotifyDispatchSkipsMissingAndCrossUserNotifications(t *testing.T) {
	db := schedulerTestDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	otherNotification := model.Notification{UserID: 2, Name: "other", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	if err := db.Create(&otherNotification).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := db.Create(&model.MonitorNotification{MonitorID: monitor.ID, NotificationID: otherNotification.ID}).Error; err != nil {
		t.Fatalf("create association: %v", err)
	}
	if err := db.Create(&model.MonitorNotification{MonitorID: monitor.ID, NotificationID: otherNotification.ID + 100}).Error; err != nil {
		t.Fatalf("create stale association: %v", err)
	}

	dispatch := NewNotifyDispatch(db)
	if err := dispatch.Send(&monitor, model.Heartbeat{MonitorID: monitor.ID, Status: model.StatusDown}, false, model.StatusUP); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestNotifyDispatchReturnsAssociationQueryErrors(t *testing.T) {
	db := schedulerTestDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Migrator().DropTable(&model.MonitorNotification{}); err != nil {
		t.Fatalf("drop associations: %v", err)
	}

	dispatch := NewNotifyDispatch(db)
	if err := dispatch.Send(&monitor, model.Heartbeat{MonitorID: monitor.ID, Status: model.StatusDown}, false, model.StatusUP); err == nil {
		t.Fatal("expected association query error")
	}
}

func TestMarkIncidentReturnsWriteErrors(t *testing.T) {
	db := schedulerTestDB(t)
	if err := db.Migrator().DropTable(&model.Incident{}); err != nil {
		t.Fatalf("drop incidents: %v", err)
	}

	dispatch := NewNotifyDispatch(db)
	if err := dispatch.markIncident(db, 1, "site", model.StatusUP, model.StatusDown, "down"); err == nil {
		t.Fatal("expected incident write error")
	}
}

func TestMarkIncidentCreatesAndResolvesIncident(t *testing.T) {
	db := schedulerTestDB(t)
	dispatch := NewNotifyDispatch(db)
	if err := dispatch.markIncident(db, 1, "site", model.StatusUP, model.StatusDown, "down"); err != nil {
		t.Fatalf("create incident: %v", err)
	}
	if err := dispatch.markIncident(db, 1, "site", model.StatusDown, model.StatusUP, "up"); err != nil {
		t.Fatalf("resolve incident: %v", err)
	}

	var incident model.Incident
	if err := db.Where("monitor_id = ?", uint(1)).First(&incident).Error; err != nil {
		t.Fatalf("find incident: %v", err)
	}
	if incident.Status != model.StatusUP || incident.EndedAt == nil || incident.Title != "site recovered" {
		t.Fatalf("incident=%+v", incident)
	}
}
