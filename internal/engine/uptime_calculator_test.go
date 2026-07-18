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
	if err := calc.CleanupOldData(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

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

func TestCleanupOldDataReturnsErrorsBeforeMutatingMemory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.StatHourly{}, &model.StatDaily{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	oldMinute := time.Now().Add(-25 * time.Hour).Unix()
	calc := NewUptimeCalculator(1, db)
	calc.MinutelyData[oldMinute] = &AggregateBucket{Up: 1}

	if err := calc.CleanupOldData(); err == nil {
		t.Fatal("expected cleanup error")
	}
	if _, ok := calc.MinutelyData[oldMinute]; !ok {
		t.Fatal("failed cleanup should not remove memory bucket")
	}
}

func TestUptimeCalculatorInitReturnsQueryErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	calc := NewUptimeCalculator(1, db)
	if err := calc.Init(); err == nil {
		t.Fatal("expected init query error")
	}
}

func TestUptimeCalculatorUpdateReturnsPersistErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.StatMinutely{}, &model.StatHourly{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	calc := NewUptimeCalculator(1, db)
	now := time.Now()
	if err := calc.Update(model.StatusUP, nil, now); err == nil {
		t.Fatal("expected persist error")
	}
	if len(calc.MinutelyData) != 0 || len(calc.HourlyData) != 0 || len(calc.DailyData) != 0 {
		t.Fatalf("failed update should not mutate memory: minute=%d hour=%d day=%d", len(calc.MinutelyData), len(calc.HourlyData), len(calc.DailyData))
	}
	var count int64
	if err := db.Model(&model.StatMinutely{}).Where("monitor_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count minutely: %v", err)
	}
	if count != 0 {
		t.Fatalf("transaction should roll back minutely rows, got %d", count)
	}
	if err := db.Model(&model.StatHourly{}).Where("monitor_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count hourly: %v", err)
	}
	if count != 0 {
		t.Fatalf("transaction should roll back hourly rows, got %d", count)
	}
}
