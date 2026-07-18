package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestUptimeRatio(t *testing.T) {
	if got := uptimeRatio(0, 0); got != 1.0 {
		t.Fatalf("empty ratio=%f want 1", got)
	}
	if got := uptimeRatio(3, 1); got != 0.75 {
		t.Fatalf("ratio=%f want 0.75", got)
	}
}

func TestDailyDataPointsUseSharedRatio(t *testing.T) {
	points := dailyDataPoints([]model.StatDaily{{
		Timestamp: 100,
		Up:        2,
		Down:      2,
		AvgPing:   42,
		MinPing:   10,
		MaxPing:   90,
	}})

	if len(points) != 1 {
		t.Fatalf("points=%d", len(points))
	}
	point := points[0]
	if point.Timestamp != 100 || point.Uptime != 0.5 || point.AvgPing != 42 || point.MinPing != 10 || point.MaxPing != 90 {
		t.Fatalf("point=%+v", point)
	}
}

func TestDailyStatsUptime(t *testing.T) {
	stats := []model.StatDaily{
		{Up: 3, Down: 1},
		{Up: 1, Down: 3},
	}
	if got := dailyStatsUptime(stats); got != 0.5 {
		t.Fatalf("uptime=%f want 0.5", got)
	}
	if got := dailyStatsUptime(nil); got != 1.0 {
		t.Fatalf("empty uptime=%f want 1", got)
	}
}

func TestUptime24HFromSLA(t *testing.T) {
	if got := uptime24HFromSLA(SLAResult{}); got != 1.0 {
		t.Fatalf("empty 24h=%f want 1", got)
	}
	if got := uptime24HFromSLA(SLAResult{UptimePercentage: 0.75, TotalChecks: 4}); got != 0.75 {
		t.Fatalf("24h=%f want 0.75", got)
	}
	if got := uptime24HFromSLA(SLAResult{UptimePercentage: 0, TotalChecks: 4}); got != 0 {
		t.Fatalf("down 24h=%f want 0", got)
	}
}
