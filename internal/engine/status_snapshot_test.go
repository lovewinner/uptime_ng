package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestPendingStatusSnapshot(t *testing.T) {
	groupID := uint(3)
	monitor := model.Monitor{
		ID:      7,
		Name:    "site",
		Type:    model.MonitorTypeHTTP,
		GroupID: &groupID,
		Active:  false,
	}

	snapshot := pendingStatusSnapshot(monitor)
	if snapshot.ID != monitor.ID || snapshot.Name != monitor.Name || snapshot.Type != monitor.Type {
		t.Fatalf("identity mismatch: %+v", snapshot)
	}
	if snapshot.GroupID == nil || *snapshot.GroupID != groupID {
		t.Fatalf("group_id=%v", snapshot.GroupID)
	}
	if snapshot.Status != model.StatusPending || snapshot.Uptime24H != 1.0 || snapshot.Active {
		t.Fatalf("defaults=%+v", snapshot)
	}
}
