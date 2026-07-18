package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/engine"
	"uptime_ng/internal/model"
)

type HeartbeatHandler struct {
	DB *gorm.DB
}

func NewHeartbeatHandler(db *gorm.DB) *HeartbeatHandler {
	return &HeartbeatHandler{DB: db}
}

func (h *HeartbeatHandler) GetBeats(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}
	period := positiveIntParam(c.DefaultQuery("period", "3600"), 3600)

	if _, err := userMonitor(h.DB, userID, monitorID); err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
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
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}
	limit := positiveIntParam(c.DefaultQuery("limit", "50"), 50)

	if _, err := userMonitor(h.DB, userID, monitorID); err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	var beats []model.Heartbeat
	h.DB.Where("monitor_id = ? AND important = ?", monitorID, true).
		Order("time DESC").
		Limit(limit).
		Find(&beats)

	c.JSON(http.StatusOK, beats)
}

func (h *HeartbeatHandler) GetIncidents(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	if _, err := userMonitor(h.DB, userID, monitorID); err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	var incidents []model.Incident
	h.DB.Where("monitor_id = ?", monitorID).
		Order("started_at DESC").
		Limit(100).
		Find(&incidents)

	now := time.Now()
	for i := range incidents {
		if incidents[i].EndedAt == nil {
			incidents[i].DurationSec = uint32(now.Sub(incidents[i].StartedAt).Seconds())
		}
	}

	c.JSON(http.StatusOK, incidents)
}

func (h *HeartbeatHandler) GetRecentStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	results, err := engine.ComputeActiveStatuses(h.DB, userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "status_query_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, results)
}

func (h *HeartbeatHandler) GetStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}
	result, err := engine.ComputeMonitorStatus(h.DB, userID, monitorID)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}
	c.JSON(http.StatusOK, result)
}
