package handler

import (
	"testing"
	"time"

	"uptime_ng/internal/model"
)

func TestMaintenanceWindowFromRequestCreatesWindow(t *testing.T) {
	start := time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	monitorID := uint(9)

	window, validationErr := maintenanceWindowFromRequest(MaintenanceRequest{
		Name:        "deploy",
		Description: "release",
		MonitorID:   &monitorID,
		StartAt:     start.Format(time.RFC3339),
		EndAt:       end.Format(time.RFC3339),
	}, 7, nil)
	if validationErr != nil {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}

	if window.UserID != 7 || window.Name != "deploy" || window.Description != "release" {
		t.Fatalf("window=%+v", window)
	}
	if window.MonitorID == nil || *window.MonitorID != monitorID {
		t.Fatalf("monitor_id=%v want %d", window.MonitorID, monitorID)
	}
	if !window.Active {
		t.Fatalf("new window should default active")
	}
	if !window.StartAt.Equal(start) || !window.EndAt.Equal(end) {
		t.Fatalf("time range=%s..%s", window.StartAt, window.EndAt)
	}
}

func TestMaintenanceWindowFromRequestPreservesExistingFields(t *testing.T) {
	start := time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC)
	existing := model.MaintenanceWindow{ID: 3, UserID: 7, Active: false}

	window, validationErr := maintenanceWindowFromRequest(MaintenanceRequest{
		Name:    "updated",
		StartAt: start.Format(time.RFC3339),
		EndAt:   start.Add(time.Hour).Format(time.RFC3339),
	}, 7, &existing)
	if validationErr != nil {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}

	if window.ID != existing.ID || window.UserID != existing.UserID {
		t.Fatalf("existing identity not preserved: %+v", window)
	}
	if window.Active {
		t.Fatalf("existing active=false should be preserved when request omits active")
	}

	window, validationErr = maintenanceWindowFromRequest(MaintenanceRequest{
		Name:    "updated",
		StartAt: start.Format(time.RFC3339),
		EndAt:   start.Add(time.Hour).Format(time.RFC3339),
		Active:  boolPtr(true),
	}, 7, &existing)
	if validationErr != nil {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}
	if !window.Active {
		t.Fatalf("request active=true should override existing active=false")
	}
}

func TestMaintenanceWindowFromRequestRejectsInvalidTimeRange(t *testing.T) {
	start := time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC)
	_, validationErr := maintenanceWindowFromRequest(MaintenanceRequest{
		Name:    "bad",
		StartAt: start.Format(time.RFC3339),
		EndAt:   start.Format(time.RFC3339),
	}, 7, nil)
	if validationErr == nil {
		t.Fatalf("expected validation error")
	}
	if validationErr.code != "invalid_time_range" {
		t.Fatalf("code=%s want invalid_time_range", validationErr.code)
	}
}

func TestBuildMaintenanceWindowValidatesOwnedMonitor(t *testing.T) {
	db := testDB(t)
	handler := NewMaintenanceHandler(db)
	other := model.Monitor{UserID: 2, Name: "other", Type: model.MonitorTypeHTTP}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	start := time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC)

	_, validationErr, lookupErr := handler.buildMaintenanceWindow(MaintenanceRequest{
		Name:      "deploy",
		MonitorID: &other.ID,
		StartAt:   start.Format(time.RFC3339),
		EndAt:     start.Add(time.Hour).Format(time.RFC3339),
	}, 1, nil)
	if lookupErr != nil {
		t.Fatalf("lookup error: %v", lookupErr)
	}
	if validationErr == nil || validationErr.code != "invalid_monitor" {
		t.Fatalf("validationErr=%+v", validationErr)
	}
}

func TestBuildMaintenanceWindowReturnsMonitorLookupErrors(t *testing.T) {
	db := testDB(t)
	handler := NewMaintenanceHandler(db)
	monitorID := uint(9)
	start := time.Date(2026, 7, 19, 1, 0, 0, 0, time.UTC)
	if err := db.Migrator().DropTable(&model.Monitor{}); err != nil {
		t.Fatalf("drop monitors: %v", err)
	}

	_, validationErr, lookupErr := handler.buildMaintenanceWindow(MaintenanceRequest{
		Name:      "deploy",
		MonitorID: &monitorID,
		StartAt:   start.Format(time.RFC3339),
		EndAt:     start.Add(time.Hour).Format(time.RFC3339),
	}, 1, nil)
	if validationErr != nil {
		t.Fatalf("validationErr=%+v", validationErr)
	}
	if lookupErr == nil {
		t.Fatal("expected lookup error")
	}
}
