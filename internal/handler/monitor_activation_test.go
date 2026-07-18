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
	monitors := []model.Monitor{{ID: 3, Active: false}, {ID: 5, Active: false}}

	if err := restartMonitors(scheduler, monitors); err != nil {
		t.Fatalf("restart monitors: %v", err)
	}
	stopMonitors(scheduler, monitors)

	got := []schedulerCall{
		{action: "restart", id: 3},
		{action: "restart", id: 5},
		{action: "stop", id: 3},
		{action: "stop", id: 5},
	}
	if len(scheduler.calls) != len(got) {
		t.Fatalf("calls=%+v want %+v", scheduler.calls, got)
	}
	for i := range got {
		if scheduler.calls[i] != got[i] {
			t.Fatalf("calls=%+v want %+v", scheduler.calls, got)
		}
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
