package handler

import "uptime_ng/internal/model"

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
