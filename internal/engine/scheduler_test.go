package engine

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestSchedulerSkipsGroupMonitors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	scheduler.StartMonitor(&model.Monitor{
		ID:       1,
		Name:     "group",
		Type:     model.MonitorTypeGroup,
		Active:   true,
		Interval: model.DefaultInterval,
	})
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
}
