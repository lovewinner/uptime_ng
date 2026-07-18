package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type SLAHandler struct {
	DB *gorm.DB
}

func NewSLAHandler(db *gorm.DB) *SLAHandler {
	return &SLAHandler{DB: db}
}

type SLARequest struct {
	PeriodType  string `json:"period_type" binding:"required"` // day, week, month, quarter, year
	PeriodStart string `json:"period_start"`                   // optional, default is current period
}

type SLAResult struct {
	MonitorID        uint    `json:"monitor_id"`
	MonitorName      string  `json:"monitor_name"`
	MonitorType      string  `json:"monitor_type"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalChecks      uint32  `json:"total_checks"`
	FailedChecks     uint32  `json:"failed_checks"`
	AvgPingMS        float64 `json:"avg_ping_ms"`
	Incidents        uint32  `json:"incidents"`
	TotalDowntimeSec uint32  `json:"total_downtime_seconds"`
}

func (h *SLAHandler) GetUptime(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var req SLARequest
	req.PeriodType = c.DefaultQuery("period", "day")
	periodStart, periodEnd := periodRange(req.PeriodType, time.Now())

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	result := SLAResult{
		MonitorID:   monitor.ID,
		MonitorName: monitor.Name,
		MonitorType: monitor.Type,
	}

	fillSLAFromHeartbeats(h.DB, &result, monitor.ID, periodStart, periodEnd)

	var incCount int64
	h.DB.Model(&model.Incident{}).
		Where("monitor_id = ? AND started_at >= ? AND started_at < ?", monitorID, periodStart, periodEnd).
		Count(&incCount)
	result.Incidents = uint32(incCount)

	c.JSON(http.StatusOK, result)
}

func (h *SLAHandler) GetOverall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	periodType := c.DefaultQuery("period", "day")
	periodStart, periodEnd := periodRange(periodType, time.Now())

	var monitors []model.Monitor
	h.DB.Where("user_id = ?", userID).Find(&monitors)

	results := make([]SLAResult, len(monitors))

	for i, m := range monitors {
		result := SLAResult{
			MonitorID:   m.ID,
			MonitorName: m.Name,
			MonitorType: m.Type,
		}
		fillSLAFromHeartbeats(h.DB, &result, m.ID, periodStart, periodEnd)
		var incCount int64
		h.DB.Model(&model.Incident{}).
			Where("monitor_id = ? AND started_at >= ? AND started_at < ?", m.ID, periodStart, periodEnd).
			Count(&incCount)
		result.Incidents = uint32(incCount)
		results[i] = result
	}

	if data, err := json.Marshal(results); err == nil {
		h.DB.Create(&model.SLAReport{
			UserID:      userID.(uint),
			PeriodType:  periodType,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			DataJSON:    string(data),
			GeneratedAt: time.Now(),
		})
	}

	c.JSON(http.StatusOK, results)
}

func (h *SLAHandler) GetUptimeData(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")
	granularity := c.DefaultQuery("granularity", "daily")
	numStr := c.DefaultQuery("num", "30")
	num := atoi(numStr)
	if num <= 0 {
		num = 30
	}

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	type dataPoint struct {
		Timestamp int64   `json:"timestamp"`
		Uptime    float64 `json:"uptime"`
		AvgPing   float64 `json:"avg_ping"`
		MinPing   float64 `json:"min_ping"`
		MaxPing   float64 `json:"max_ping"`
		Up        uint32  `json:"up"`
		Down      uint32  `json:"down"`
	}

	switch granularity {
	case "minutely":
		var stats []model.StatMinutely
		cutoff := timeNowUnix() - int64(num)*60
		h.DB.Where("monitor_id = ? AND timestamp >= ?", monitorID, cutoff).Order("timestamp ASC").Find(&stats)
		points := make([]dataPoint, len(stats))
		for i, s := range stats {
			pts := s.Up + s.Down
			var uptime float64 = 1.0
			if pts > 0 {
				uptime = float64(s.Up) / float64(pts)
			}
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, MinPing: s.MinPing, MaxPing: s.MaxPing, Up: s.Up, Down: s.Down}
		}
		c.JSON(http.StatusOK, points)

	case "hourly":
		var stats []model.StatHourly
		cutoff := timeNowUnix() - int64(num)*3600
		h.DB.Where("monitor_id = ? AND timestamp >= ?", monitorID, cutoff).Order("timestamp ASC").Find(&stats)
		points := make([]dataPoint, len(stats))
		for i, s := range stats {
			pts := s.Up + s.Down
			var uptime float64 = 1.0
			if pts > 0 {
				uptime = float64(s.Up) / float64(pts)
			}
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, MinPing: s.MinPing, MaxPing: s.MaxPing, Up: s.Up, Down: s.Down}
		}
		c.JSON(http.StatusOK, points)

	default:
		var stats []model.StatDaily
		cutoff := timeNowUnix() - int64(num)*86400
		h.DB.Where("monitor_id = ? AND timestamp >= ?", monitorID, cutoff).Order("timestamp ASC").Find(&stats)
		points := make([]dataPoint, len(stats))
		for i, s := range stats {
			pts := s.Up + s.Down
			var uptime float64 = 1.0
			if pts > 0 {
				uptime = float64(s.Up) / float64(pts)
			}
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, MinPing: s.MinPing, MaxPing: s.MaxPing, Up: s.Up, Down: s.Down}
		}
		c.JSON(http.StatusOK, points)
	}
}

func (h *SLAHandler) GetUptimeSummary(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	type uptimeSummary struct {
		Uptime24H float64 `json:"uptime_24h"`
		Uptime30D float64 `json:"uptime_30d"`
		Uptime1Y  float64 `json:"uptime_1y"`
	}

	var result uptimeSummary

	// 24h: reuse fillSLAFromHeartbeats for consistency
	sla24 := SLAResult{}
	fillSLAFromHeartbeats(h.DB, &sla24, monitor.ID, time.Now().Add(-24*time.Hour), time.Now())
	result.Uptime24H = sla24.UptimePercentage
	if result.Uptime24H == 0 && sla24.TotalChecks == 0 {
		result.Uptime24H = 1.0
	}

	// 30d / 1y: compute from stat_dailies
	computeDaily := func(days int) float64 {
		cutoff := timeNowUnix() - int64(days)*86400
		var stats []model.StatDaily
		h.DB.Where("monitor_id = ? AND timestamp >= ?", monitorID, cutoff).Find(&stats)
		var up, down uint32
		for _, s := range stats {
			up += s.Up
			down += s.Down
		}
		total := up + down
		if total > 0 {
			return float64(up) / float64(total)
		}
		return 1.0
	}
	result.Uptime30D = computeDaily(30)
	result.Uptime1Y = computeDaily(365)

	c.JSON(http.StatusOK, result)
}

func periodToDays(period string) int {
	switch period {
	case "day":
		return 1
	case "week":
		return 7
	case "month":
		return 30
	case "quarter":
		return 90
	case "year":
		return 365
	default:
		return 1
	}
}

func periodRange(period string, now time.Time) (time.Time, time.Time) {
	loc := now.Location()
	year, month, day := now.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, loc)

	switch period {
	case "week":
		weekday := int(start.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = start.AddDate(0, 0, -(weekday - 1))
		return start, start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(year, month, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0)
	case "quarter":
		quarterMonth := time.Month(((int(month)-1)/3)*3 + 1)
		start = time.Date(year, quarterMonth, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	case "year":
		start = time.Date(year, 1, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0)
	default:
		return start, start.AddDate(0, 0, 1)
	}
}

func fillSLAFromHeartbeats(db *gorm.DB, result *SLAResult, monitorID uint, start, end time.Time) {
	var previous model.Heartbeat
	db.Where("monitor_id = ? AND time < ?", monitorID, start).Order("time DESC").First(&previous)

	status := model.StatusUP
	if previous.ID > 0 {
		status = previous.Status
	}

	var beats []model.Heartbeat
	db.Where("monitor_id = ? AND time >= ? AND time < ?", monitorID, start, end).
		Order("time ASC").
		Find(&beats)

	last := start
	var downtime time.Duration
	var pingTotal float64
	var pingCount uint32

	for _, beat := range beats {
		if beat.Status == model.StatusPending {
			continue
		}
		if beat.Time.After(last) && model.FlatStatus(status) == model.StatusDown {
			downtime += beat.Time.Sub(last)
		}
		result.TotalChecks++
		if model.FlatStatus(beat.Status) == model.StatusDown {
			result.FailedChecks++
		}
		if model.FlatStatus(beat.Status) == model.StatusUP && beat.PingMS != nil {
			pingTotal += *beat.PingMS
			pingCount++
		}
		status = beat.Status
		last = beat.Time
	}

	if end.After(last) && model.FlatStatus(status) == model.StatusDown {
		downtime += end.Sub(last)
	}

	duration := end.Sub(start)
	result.UptimePercentage = 1.0
	if duration > 0 {
		result.UptimePercentage = 1 - downtime.Seconds()/duration.Seconds()
		if result.UptimePercentage < 0 {
			result.UptimePercentage = 0
		}
	}
	if pingCount > 0 {
		result.AvgPingMS = pingTotal / float64(pingCount)
	}
	result.TotalDowntimeSec = uint32(downtime.Seconds())
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
