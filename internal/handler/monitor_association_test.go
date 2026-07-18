package handler

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestRefreshMonitorAssociationsHonorsIndependentFlags(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	oldNotif := model.Notification{UserID: 1, Name: "old", Type: model.NotificationTypeEmail, Config: `{"to":"old@example.com"}`, Active: true}
	newNotif := model.Notification{UserID: 1, Name: "new", Type: model.NotificationTypeEmail, Config: `{"to":"new@example.com"}`, Active: true}
	if err := db.Create(&oldNotif).Error; err != nil {
		t.Fatalf("create old notif: %v", err)
	}
	if err := db.Create(&newNotif).Error; err != nil {
		t.Fatalf("create new notif: %v", err)
	}

	if err := runTransaction(db, func(tx *gorm.DB) error {
		if err := attachMonitorAssociations(tx, monitor.ID, []uint{oldNotif.ID}, []string{"prod"}, []string{"#123456"}); err != nil {
			return err
		}
		return refreshMonitorAssociations(tx, monitor.ID, []uint{newNotif.ID}, nil, nil, true, false)
	}); err != nil {
		t.Fatalf("refresh associations: %v", err)
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

func TestAttachExportMonitorAssociationsUsesOwnedNotifications(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP}
	ownedNotif := model.Notification{UserID: 1, Name: "ops", Type: model.NotificationTypeEmail, Config: `{"to":"own@example.com"}`, Active: true}
	otherNotif := model.Notification{UserID: 2, Name: "ops", Type: model.NotificationTypeEmail, Config: `{"to":"other@example.com"}`, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Create(&otherNotif).Error; err != nil {
		t.Fatalf("create other notif: %v", err)
	}
	if err := db.Create(&ownedNotif).Error; err != nil {
		t.Fatalf("create owned notif: %v", err)
	}

	if err := runTransaction(db, func(tx *gorm.DB) error {
		return attachExportMonitorAssociations(tx, 1, monitor.ID, ExportMonitor{
			Tags:              []ExportTag{{Name: "prod", Color: ""}},
			NotificationNames: []string{"ops"},
		})
	}); err != nil {
		t.Fatalf("attach export associations: %v", err)
	}

	var links []model.MonitorNotification
	db.Where("monitor_id = ?", monitor.ID).Find(&links)
	if len(links) != 1 || links[0].NotificationID != ownedNotif.ID {
		t.Fatalf("notification links=%+v want owned=%d", links, ownedNotif.ID)
	}

	var tags []model.MonitorTag
	db.Where("monitor_id = ?", monitor.ID).Find(&tags)
	if len(tags) != 1 {
		t.Fatalf("tags=%+v want one tag", tags)
	}
}

func TestAttachExportMonitorAssociationsReturnsLookupErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("lookup failed")
	tx := db.Session(&gorm.Session{DryRun: true})
	tx.AddError(wantErr)

	err := attachExportMonitorAssociations(tx, 1, 1, ExportMonitor{
		Tags:              []ExportTag{{Name: "prod", Color: ""}},
		NotificationNames: []string{"ops"},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestDeleteMonitorDataRemovesOwnedDependentsAndUngroupsChildren(t *testing.T) {
	db := testDB(t)
	parent := model.Monitor{UserID: 1, Name: "parent", Type: model.MonitorTypeGroup}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeHTTP, GroupID: &parent.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	sibling := model.Monitor{UserID: 1, Name: "sibling", Type: model.MonitorTypeHTTP}
	if err := db.Create(&sibling).Error; err != nil {
		t.Fatalf("create sibling: %v", err)
	}
	tag := model.Tag{Name: "prod", Color: "#123456"}
	orphanTag := model.Tag{Name: "orphan", Color: "#654321"}
	notif := model.Notification{UserID: 1, Name: "ops", Type: model.NotificationTypeEmail, Config: `{}`, Active: true}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := db.Create(&orphanTag).Error; err != nil {
		t.Fatalf("create orphan tag: %v", err)
	}
	if err := db.Create(&notif).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	now := time.Now()
	fixtures := []any{
		&model.Heartbeat{MonitorID: parent.ID, Time: now},
		&model.MonitorNotification{MonitorID: parent.ID, NotificationID: notif.ID},
		&model.MonitorTag{MonitorID: parent.ID, TagID: tag.ID, Value: tag.Name},
		&model.MonitorTag{MonitorID: sibling.ID, TagID: tag.ID, Value: tag.Name},
		&model.MonitorTag{MonitorID: parent.ID, TagID: orphanTag.ID, Value: orphanTag.Name},
		&model.MaintenanceWindow{UserID: 1, MonitorID: &parent.ID, Name: "window", StartAt: now, EndAt: now.Add(time.Hour)},
		&model.StatMinutely{MonitorID: parent.ID, Timestamp: now.Unix()},
		&model.StatHourly{MonitorID: parent.ID, Timestamp: now.Unix()},
		&model.StatDaily{MonitorID: parent.ID, Timestamp: now.Unix()},
		&model.Incident{MonitorID: parent.ID, StartedAt: now},
	}
	for _, fixture := range fixtures {
		if err := db.Create(fixture).Error; err != nil {
			t.Fatalf("create fixture %T: %v", fixture, err)
		}
	}

	if err := runTransaction(db, func(tx *gorm.DB) error {
		return deleteMonitorData(tx, parent)
	}); err != nil {
		t.Fatalf("deleteMonitorData: %v", err)
	}

	var count int64
	db.Model(&model.Monitor{}).Where("id = ?", parent.ID).Count(&count)
	if count != 0 {
		t.Fatalf("monitor count=%d want 0", count)
	}
	for _, modelValue := range []any{
		&model.Heartbeat{}, &model.MonitorNotification{}, &model.MonitorTag{},
		&model.MaintenanceWindow{}, &model.StatMinutely{}, &model.StatHourly{}, &model.StatDaily{}, &model.Incident{},
	} {
		db.Model(modelValue).Where("monitor_id = ?", parent.ID).Count(&count)
		if count != 0 {
			t.Fatalf("%T count=%d want 0", modelValue, count)
		}
	}
	var updatedChild model.Monitor
	if err := db.First(&updatedChild, child.ID).Error; err != nil {
		t.Fatalf("load child: %v", err)
	}
	if updatedChild.GroupID != nil {
		t.Fatalf("child group_id=%v want nil", *updatedChild.GroupID)
	}
	db.Model(&model.Tag{}).Where("id = ?", tag.ID).Count(&count)
	if count != 1 {
		t.Fatalf("shared tag count=%d want 1", count)
	}
	db.Model(&model.Tag{}).Where("id = ?", orphanTag.ID).Count(&count)
	if count != 0 {
		t.Fatalf("orphan tag count=%d want 0", count)
	}
}

func TestUngroupChildMonitorsClearsDirectChildrenOnly(t *testing.T) {
	db := testDB(t)
	parent := model.Monitor{UserID: 1, Name: "parent", Type: model.MonitorTypeGroup}
	otherParent := model.Monitor{UserID: 1, Name: "other-parent", Type: model.MonitorTypeGroup}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := db.Create(&otherParent).Error; err != nil {
		t.Fatalf("create other parent: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeHTTP, GroupID: &parent.ID}
	otherChild := model.Monitor{UserID: 1, Name: "other-child", Type: model.MonitorTypeHTTP, GroupID: &otherParent.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := db.Create(&otherChild).Error; err != nil {
		t.Fatalf("create other child: %v", err)
	}

	if err := runTransaction(db, func(tx *gorm.DB) error {
		return ungroupChildMonitors(tx, parent.ID)
	}); err != nil {
		t.Fatalf("ungroupChildMonitors: %v", err)
	}

	var updatedChild model.Monitor
	var updatedOtherChild model.Monitor
	db.First(&updatedChild, child.ID)
	db.First(&updatedOtherChild, otherChild.ID)
	if updatedChild.GroupID != nil {
		t.Fatalf("child group_id=%v want nil", *updatedChild.GroupID)
	}
	if updatedOtherChild.GroupID == nil || *updatedOtherChild.GroupID != otherParent.ID {
		t.Fatalf("other child group_id=%v want %d", updatedOtherChild.GroupID, otherParent.ID)
	}
}
