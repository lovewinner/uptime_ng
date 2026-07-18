package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestApplyCheckHeartbeatStateRetriesDownResultAsPending(t *testing.T) {
	previous := model.Heartbeat{
		ID:        1,
		Status:    model.StatusUP,
		Retries:   0,
		DownCount: 0,
	}
	beat := model.Heartbeat{Status: model.StatusDown}
	transition := applyCheckHeartbeatState(&beat, previous, &model.Monitor{
		MaxRetries: 2,
	}, &CheckResult{Status: model.StatusDown})

	if transition.isFirstBeat || transition.previousStatus != model.StatusUP {
		t.Fatalf("transition=%+v", transition)
	}
	if beat.Status != model.StatusPending {
		t.Fatalf("status=%d want pending", beat.Status)
	}
	if beat.Retries != 1 {
		t.Fatalf("retries=%d want 1", beat.Retries)
	}
	if !beat.Important {
		t.Fatal("UP -> PENDING should be important")
	}
	if beat.DownCount != 0 {
		t.Fatalf("important beat should reset down_count, got %d", beat.DownCount)
	}
}

func TestApplyCheckHeartbeatStateHonorsRetryOnlyOnStatusCode(t *testing.T) {
	previous := model.Heartbeat{ID: 1, Status: model.StatusDown, Retries: 0, DownCount: 3}
	beat := model.Heartbeat{Status: model.StatusDown}
	applyCheckHeartbeatState(&beat, previous, &model.Monitor{
		MaxRetries:            2,
		RetryOnlyOnStatusCode: true,
	}, &CheckResult{Status: model.StatusDown, HTTPStatus: 0})

	if beat.Status != model.StatusDown {
		t.Fatalf("status=%d want down", beat.Status)
	}
	if beat.Retries != 0 {
		t.Fatalf("retries=%d want 0", beat.Retries)
	}
	if beat.DownCount != 4 {
		t.Fatalf("down_count=%d want 4", beat.DownCount)
	}
	if beat.Important {
		t.Fatal("DOWN -> DOWN should not be important")
	}
}

func TestApplyGroupHeartbeatStateClearsCountersWhenUp(t *testing.T) {
	previous := model.Heartbeat{ID: 1, Status: model.StatusDown, Retries: 2, DownCount: 5}
	beat := model.Heartbeat{Status: model.StatusUP}
	transition := applyGroupHeartbeatState(&beat, previous)

	if transition.previousStatus != model.StatusDown {
		t.Fatalf("previous status=%d", transition.previousStatus)
	}
	if !beat.Important {
		t.Fatal("DOWN -> UP should be important")
	}
	if beat.Retries != 0 || beat.DownCount != 0 {
		t.Fatalf("counters retries=%d down_count=%d", beat.Retries, beat.DownCount)
	}
}
