package handler

import (
	"net/http"
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
	days := periodToDays(req.PeriodType)

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	result := SLAResult{
		MonitorID:   monitor.ID,
		MonitorName: monitor.Name,
		MonitorType: monitor.Type,
	}

	var stats []model.StatDaily
	cutoff := int64(0)
	if days > 0 {
		cutoff = timeNowUnix() - int64(days)*86400
	}

	query := h.DB.Where("monitor_id = ?", monitorID)
	if cutoff > 0 {
		query = query.Where("timestamp >= ?", cutoff)
	}
	query.Find(&stats)

	var totalUP, totalDown uint32
	var totalPing float64
	for _, s := range stats {
		totalUP += s.Up
		totalDown += s.Down
		totalPing += s.AvgPing * float64(s.Up)
	}
	result.TotalChecks = totalUP + totalDown
	result.FailedChecks = totalDown
	if result.TotalChecks > 0 {
		result.UptimePercentage = float64(totalUP) / float64(result.TotalChecks)
	} else {
		result.UptimePercentage = 1.0
	}
	if totalUP > 0 {
		result.AvgPingMS = totalPing / float64(totalUP)
	}

	var incCount int64
	h.DB.Model(&model.Incident{}).Where("monitor_id = ?", monitorID).Where("started_at >= to_timestamp(?)", cutoff).Count(&incCount)
	result.Incidents = uint32(incCount)

	c.JSON(http.StatusOK, result)
}

func (h *SLAHandler) GetOverall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	periodType := c.DefaultQuery("period", "day")
	days := periodToDays(periodType)

	var monitors []model.Monitor
	h.DB.Where("user_id = ?", userID).Find(&monitors)

	results := make([]SLAResult, len(monitors))
	cutoff := timeNowUnix() - int64(days)*86400

	for i, m := range monitors {
		var stats []model.StatDaily
		h.DB.Where("monitor_id = ? AND timestamp >= ?", m.ID, cutoff).Find(&stats)

		var totalUP, totalDown uint32
		var totalPing float64
		for _, s := range stats {
			totalUP += s.Up
			totalDown += s.Down
			totalPing += s.AvgPing * float64(s.Up)
		}
		result := SLAResult{
			MonitorID:     m.ID,
			MonitorName:   m.Name,
			MonitorType:   m.Type,
			TotalChecks:   totalUP + totalDown,
			FailedChecks:  totalDown,
		}
		if result.TotalChecks > 0 {
			result.UptimePercentage = float64(totalUP) / float64(result.TotalChecks)
		} else {
			result.UptimePercentage = 1.0
		}
		if totalUP > 0 {
			result.AvgPingMS = totalPing / float64(totalUP)
		}
		results[i] = result
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
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	type dataPoint struct {
		Timestamp int64   `json:"timestamp"`
		Uptime    float64 `json:"uptime"`
		AvgPing   float64 `json:"avg_ping"`
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
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, Up: s.Up, Down: s.Down}
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
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, Up: s.Up, Down: s.Down}
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
			points[i] = dataPoint{Timestamp: s.Timestamp, Uptime: uptime, AvgPing: s.AvgPing, Up: s.Up, Down: s.Down}
		}
		c.JSON(http.StatusOK, points)
	}
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

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}