package engine

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestCleanupOldDataRemovesDailyStats(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.StatMinutely{}, &model.StatHourly{}, &model.StatDaily{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	oldDaily := time.Now().Add(-366 * 24 * time.Hour).Unix()
	recentDaily := time.Now().Add(-10 * 24 * time.Hour).Unix()
	db.Create(&model.StatDaily{MonitorID: 1, Timestamp: oldDaily, Up: 1})
	db.Create(&model.StatDaily{MonitorID: 1, Timestamp: recentDaily, Up: 1})

	calc := NewUptimeCalculator(1, db)
	calc.DailyData[oldDaily] = &AggregateBucket{Up: 1}
	calc.DailyData[recentDaily] = &AggregateBucket{Up: 1}
	calc.CleanupOldData()

	var count int64
	db.Model(&model.StatDaily{}).Where("monitor_id = ?", 1).Count(&count)
	if count != 1 {
		t.Fatalf("remaining daily rows=%d want 1", count)
	}
	if _, ok := calc.DailyData[oldDaily]; ok {
		t.Fatalf("old daily bucket still in memory")
	}
	if _, ok := calc.DailyData[recentDaily]; !ok {
		t.Fatalf("recent daily bucket removed")
	}
}
