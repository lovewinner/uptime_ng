package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestComputeMonitorStatusDefaultsWhenHeartbeatMissing(t *testing.T) {
	db := schedulerTestDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	status, err := ComputeMonitorStatus(db, monitor.UserID, monitor.ID)
	if err != nil {
		t.Fatalf("compute status: %v", err)
	}
	if status.Status != model.StatusDown || status.Uptime24H != 1.0 {
		t.Fatalf("status=%+v", status)
	}
}

func TestComputeMonitorStatusReturnsHeartbeatQueryErrors(t *testing.T) {
	db := schedulerTestDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Migrator().DropTable(&model.Heartbeat{}); err != nil {
		t.Fatalf("drop heartbeats: %v", err)
	}

	if _, err := ComputeMonitorStatus(db, monitor.UserID, monitor.ID); err == nil {
		t.Fatal("expected heartbeat query error")
	}
}

func TestComputeGroupStatusReturnsChildQueryErrors(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := db.Migrator().DropTable(&model.Monitor{}); err != nil {
		t.Fatalf("drop monitors: %v", err)
	}

	if _, err := computeMonitorStatus(db, group, map[uint]bool{}); err == nil {
		t.Fatal("expected child query error")
	}
}
