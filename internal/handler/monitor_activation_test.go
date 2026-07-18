package handler

import (
	"errors"
	"testing"

	"uptime_ng/internal/model"
)

func TestMonitorActivationTargets(t *testing.T) {
	leaf := model.Monitor{ID: 1, Type: model.MonitorTypeHTTP}
	descendants := []model.Monitor{{ID: 2, Type: model.MonitorTypeHTTP}}
	targets := monitorActivationTargets(leaf, descendants)
	if len(targets) != 1 || targets[0].ID != leaf.ID {
		t.Fatalf("leaf targets=%+v", targets)
	}

	group := model.Monitor{ID: 10, Type: model.MonitorTypeGroup}
	targets = monitorActivationTargets(group, descendants)
	if len(targets) != 2 || targets[0].ID != group.ID || targets[1].ID != descendants[0].ID {
		t.Fatalf("group targets=%+v", targets)
	}
}

func TestMonitorIDs(t *testing.T) {
	ids := monitorIDs([]model.Monitor{{ID: 3}, {ID: 5}})
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 5 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestRestartAndStopMonitors(t *testing.T) {
	scheduler := &fakeScheduler{}
	monitors := []model.Monitor{{ID: 3, Type: model.MonitorTypeHTTP, Active: false}, {ID: 5, Type: model.MonitorTypePush, Active: false}}

	if err := restartMonitors(scheduler, monitors); err != nil {
		t.Fatalf("restart monitors: %v", err)
	}
	stopMonitors(scheduler, monitors)

	got := []schedulerCall{
		{action: "restart", id: 3},
		{action: "stop", id: 5},
		{action: "stop", id: 3},
		{action: "stop", id: 5},
	}
	assertSchedulerCalls(t, scheduler, got)
}

func TestShouldRunScheduledMonitor(t *testing.T) {
	tests := []struct {
		name    string
		monitor model.Monitor
		want    bool
	}{
		{name: "active http", monitor: model.Monitor{Type: model.MonitorTypeHTTP, Active: true}, want: true},
		{name: "active group", monitor: model.Monitor{Type: model.MonitorTypeGroup, Active: true}, want: true},
		{name: "active push", monitor: model.Monitor{Type: model.MonitorTypePush, Active: true}, want: false},
		{name: "inactive http", monitor: model.Monitor{Type: model.MonitorTypeHTTP, Active: false}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRunScheduledMonitor(tt.monitor); got != tt.want {
				t.Fatalf("shouldRunScheduledMonitor=%v want %v", got, tt.want)
			}
		})
	}
}

func TestRestartMonitorsReturnsSchedulerErrors(t *testing.T) {
	wantErr := errors.New("restart failed")
	scheduler := &fakeScheduler{restartErr: wantErr}
	err := restartMonitors(scheduler, []model.Monitor{{ID: 3, Active: false}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestRestartMonitorsDoesNotMutateInputs(t *testing.T) {
	scheduler := &fakeScheduler{}
	monitors := []model.Monitor{{ID: 3, Active: false}}
	if err := restartMonitors(scheduler, monitors); err != nil {
		t.Fatalf("restart monitors: %v", err)
	}
	if monitors[0].Active {
		t.Fatal("restartMonitors should not mutate input monitor active state")
	}
}

func TestSetMonitorActivationRollsBackResumeOnSchedulerError(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: false}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("active", false).Error; err != nil {
		t.Fatalf("deactivate monitor: %v", err)
	}
	monitor.Active = false
	alreadyActive := model.Monitor{UserID: 1, Name: "api", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&alreadyActive).Error; err != nil {
		t.Fatalf("create active monitor: %v", err)
	}

	wantErr := errors.New("restart failed")
	scheduler := &fakeScheduler{restartErrFor: map[uint]error{monitor.ID: wantErr}}
	err := setMonitorActivation(db, scheduler, []model.Monitor{monitor, alreadyActive}, true)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}

	var stored model.Monitor
	if err := db.First(&stored, monitor.ID).Error; err != nil {
		t.Fatalf("reload monitor: %v", err)
	}
	if stored.Active {
		t.Fatal("monitor active should be rolled back after scheduler failure")
	}
	var storedActive model.Monitor
	if err := db.First(&storedActive, alreadyActive.ID).Error; err != nil {
		t.Fatalf("reload active monitor: %v", err)
	}
	if !storedActive.Active {
		t.Fatal("already-active monitor should keep its original active state after rollback")
	}

	wantCalls := []schedulerCall{
		{action: "restart", id: monitor.ID},
		{action: "restart", id: alreadyActive.ID},
		{action: "stop", id: monitor.ID},
		{action: "stop", id: alreadyActive.ID},
		{action: "restart", id: alreadyActive.ID},
	}
	assertSchedulerCalls(t, scheduler, wantCalls)
}

func TestSetMonitorActivationDeactivatesOriginalActiveWhenRestoreFails(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	wantErr := errors.New("restart failed")
	scheduler := &fakeScheduler{restartErr: wantErr}
	err := setMonitorActivation(db, scheduler, []model.Monitor{monitor}, true)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}

	var stored model.Monitor
	if err := db.First(&stored, monitor.ID).Error; err != nil {
		t.Fatalf("reload monitor: %v", err)
	}
	if stored.Active {
		t.Fatal("monitor should be inactive when original active restore also fails")
	}

	wantCalls := []schedulerCall{
		{action: "restart", id: monitor.ID},
		{action: "stop", id: monitor.ID},
		{action: "restart", id: monitor.ID},
		{action: "stop", id: monitor.ID},
	}
	assertSchedulerCalls(t, scheduler, wantCalls)
}
