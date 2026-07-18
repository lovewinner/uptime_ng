package handler

import "uptime_ng/internal/model"

type uptimeSummary struct {
	Uptime24H float64 `json:"uptime_24h"`
	Uptime30D float64 `json:"uptime_30d"`
	Uptime1Y  float64 `json:"uptime_1y"`
}

func uptimeRatio(up uint32, down uint32) float64 {
	total := up + down
	if total == 0 {
		return 1.0
	}
	return float64(up) / float64(total)
}

func minutelyDataPoints(stats []model.StatMinutely) []uptimeDataPoint {
	points := make([]uptimeDataPoint, len(stats))
	for i, stat := range stats {
		points[i] = uptimeDataPointFromStats(stat.Timestamp, stat.Up, stat.Down, stat.AvgPing, stat.MinPing, stat.MaxPing)
	}
	return points
}

func hourlyDataPoints(stats []model.StatHourly) []uptimeDataPoint {
	points := make([]uptimeDataPoint, len(stats))
	for i, stat := range stats {
		points[i] = uptimeDataPointFromStats(stat.Timestamp, stat.Up, stat.Down, stat.AvgPing, stat.MinPing, stat.MaxPing)
	}
	return points
}

func dailyDataPoints(stats []model.StatDaily) []uptimeDataPoint {
	points := make([]uptimeDataPoint, len(stats))
	for i, stat := range stats {
		points[i] = uptimeDataPointFromStats(stat.Timestamp, stat.Up, stat.Down, stat.AvgPing, stat.MinPing, stat.MaxPing)
	}
	return points
}

func dailyStatsUptime(stats []model.StatDaily) float64 {
	var up, down uint32
	for _, stat := range stats {
		up += stat.Up
		down += stat.Down
	}
	return uptimeRatio(up, down)
}

func uptime24HFromSLA(result SLAResult) float64 {
	if result.UptimePercentage == 0 && result.TotalChecks == 0 {
		return 1.0
	}
	return result.UptimePercentage
}

func uptimeDataPointFromStats(timestamp int64, up uint32, down uint32, avgPing float64, minPing float64, maxPing float64) uptimeDataPoint {
	return uptimeDataPoint{
		Timestamp: timestamp,
		Uptime:    uptimeRatio(up, down),
		AvgPing:   avgPing,
		MinPing:   minPing,
		MaxPing:   maxPing,
		Up:        up,
		Down:      down,
	}
}
