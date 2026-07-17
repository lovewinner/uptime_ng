package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type HeartbeatHandler struct {
	DB *gorm.DB
}

func NewHeartbeatHandler(db *gorm.DB) *HeartbeatHandler {
	return &HeartbeatHandler{DB: db}
}

func (h *HeartbeatHandler) GetBeats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")
	period := parseInt(c.DefaultQuery("period", "3600")) // default 1h in seconds
	if period <= 0 {
		period = 3600
	}

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	var beats []model.Heartbeat
	cutoff := time.Now().Add(-time.Duration(period) * time.Second)
	h.DB.Where("monitor_id = ?", monitorID).
		Where("time > ?", cutoff).
		Order("time ASC").
		Find(&beats)

	c.JSON(http.StatusOK, beats)
}

func (h *HeartbeatHandler) GetImportantBeats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")
	limit := c.DefaultQuery("limit", "50")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	var beats []model.Heartbeat
	h.DB.Where("monitor_id = ? AND important = ?", monitorID, true).
		Order("time DESC").
		Limit(parseInt(limit)).
		Find(&beats)

	c.JSON(http.StatusOK, beats)
}

func (h *HeartbeatHandler) GetIncidents(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	var incidents []model.Incident
	h.DB.Where("monitor_id = ?", monitorID).
		Order("started_at DESC").
		Limit(100).
		Find(&incidents)

	c.JSON(http.StatusOK, incidents)
}

func (h *HeartbeatHandler) GetRecentStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	type statusItem struct {
		ID        uint    `json:"id"`
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Status    uint16  `json:"status"`
		PingMS    float64 `json:"ping_ms"`
		Uptime24H float64 `json:"uptime_24h"`
		Active    bool    `json:"active"`
	}

	var monitors []model.Monitor
	h.DB.Where("user_id = ? AND active = ?", userID, true).Find(&monitors)

	results := make([]statusItem, len(monitors))
	for i, m := range monitors {
		results[i] = statusItem{
			ID:     m.ID,
			Name:   m.Name,
			Type:   m.Type,
			Status: model.StatusDown,
			Active: true,
		}

		var beat model.Heartbeat
		if err := h.DB.Where("monitor_id = ?", m.ID).Order("time DESC").First(&beat).Error; err == nil {
			results[i].Status = beat.Status
			if beat.PingMS != nil {
				results[i].PingMS = *beat.PingMS
			}
		}

		var up, down int64
		cutoff := time.Now().Add(-24 * time.Hour)
		h.DB.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", m.ID, cutoff, model.StatusUP).Count(&up)
		h.DB.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", m.ID, cutoff, model.StatusDown).Count(&down)
		if up+down > 0 {
			results[i].Uptime24H = float64(up) / float64(up+down)
		} else {
			results[i].Uptime24H = 1.0
		}
	}

	c.JSON(http.StatusOK, results)
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
