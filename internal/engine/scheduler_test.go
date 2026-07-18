package engine

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestSchedulerRunsGroupMonitors(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	scheduler.StartMonitor(&group)
	defer scheduler.StopAll()
	if scheduler.RunningCount() != 1 {
		t.Fatalf("running=%d want 1", scheduler.RunningCount())
	}
}

func TestGroupBeatPersistsMaintenanceStatus(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	db.Create(&group)
	db.Create(&model.MaintenanceWindow{
		UserID:  1,
		Name:    "planned",
		StartAt: time.Now().Add(-time.Hour),
		EndAt:   time.Now().Add(time.Hour),
		Active:  true,
	})
	runner := &MonitorRunner{Monitor: &group, Calculator: NewUptimeCalculator(group.ID, db), DB: db}
	if err := runner.Calculator.Init(); err != nil {
		t.Fatalf("calc init: %v", err)
	}
	runner.beat()
	var beat model.Heartbeat
	if err := db.Where("monitor_id = ?", group.ID).First(&beat).Error; err != nil {
		t.Fatalf("heartbeat missing: %v", err)
	}
	if beat.Status != model.StatusMaintenance {
		t.Fatalf("status=%d want maintenance", beat.Status)
	}
}

func TestGroupBeatPersistsAggregatedStatus(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeHTTP, GroupID: &group.ID, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := db.Create(&model.Heartbeat{MonitorID: child.ID, Status: model.StatusUP, Time: time.Now()}).Error; err != nil {
		t.Fatalf("create child heartbeat: %v", err)
	}

	runner := &MonitorRunner{Monitor: &group, Calculator: NewUptimeCalculator(group.ID, db), DB: db}
	if err := runner.Calculator.Init(); err != nil {
		t.Fatalf("calc init: %v", err)
	}
	runner.beat()

	var beat model.Heartbeat
	if err := db.Where("monitor_id = ?", group.ID).Order("time DESC").First(&beat).Error; err != nil {
		t.Fatalf("group heartbeat missing: %v", err)
	}
	if beat.Status != model.StatusUP {
		t.Fatalf("status=%d want UP", beat.Status)
	}
}

func schedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Monitor{},
		&model.Heartbeat{},
		&model.StatMinutely{},
		&model.StatHourly{},
		&model.StatDaily{},
		&model.MaintenanceWindow{},
		&model.Notification{},
		&model.MonitorNotification{},
		&model.Incident{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
