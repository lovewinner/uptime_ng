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

type monitorStatusItem struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	GroupID   *uint   `json:"group_id"`
	Status    uint16  `json:"status"`
	PingMS    float64 `json:"ping_ms"`
	Uptime24H float64 `json:"uptime_24h"`
	Active    bool    `json:"active"`
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
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")
	limit := c.DefaultQuery("limit", "50")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
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
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
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

	var monitors []model.Monitor
	h.DB.Where("user_id = ? AND active = ?", userID, true).Find(&monitors)

	results := make([]monitorStatusItem, 0, len(monitors))
	resultByID := make(map[uint]*monitorStatusItem, len(monitors))
	childrenByGroup := make(map[uint][]uint)
	monitorByID := make(map[uint]model.Monitor, len(monitors))

	for _, m := range monitors {
		m := m
		item := monitorStatusItem{
			ID:      m.ID,
			Name:    m.Name,
			Type:    m.Type,
			GroupID: m.GroupID,
			Status:  model.StatusPending,
			Active:  true,
		}
		if m.Type != model.MonitorTypeGroup {
			item.Status = model.StatusDown
		}
		var beat model.Heartbeat
		if m.Type != model.MonitorTypeGroup && h.DB.Where("monitor_id = ?", m.ID).Order("time DESC").First(&beat).Error == nil {
			item.Status = beat.Status
			if beat.PingMS != nil {
				item.PingMS = *beat.PingMS
			}
		}

		if m.Type != model.MonitorTypeGroup {
			item.Uptime24H = h.monitorUptime24H(m.ID)
		} else {
			item.Uptime24H = 1.0
		}
		results = append(results, item)
		resultByID[m.ID] = &results[len(results)-1]
		monitorByID[m.ID] = m
		if m.GroupID != nil {
			childrenByGroup[*m.GroupID] = append(childrenByGroup[*m.GroupID], m.ID)
		}
	}

	for _, m := range monitors {
		if m.Type == model.MonitorTypeGroup {
			status, uptime := aggregateGroupStatus(m.ID, resultByID, monitorByID, childrenByGroup, map[uint]bool{})
			resultByID[m.ID].Status = status
			resultByID[m.ID].Uptime24H = uptime
			resultByID[m.ID].PingMS = 0
		}
	}

	c.JSON(http.StatusOK, results)
}

func (h *HeartbeatHandler) monitorUptime24H(monitorID uint) float64 {
	var up, down int64
	cutoff := time.Now().Add(-24 * time.Hour)
	h.DB.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusUP).Count(&up)
	h.DB.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusDown).Count(&down)
	if up+down > 0 {
		return float64(up) / float64(up+down)
	}
	return 1.0
}

func aggregateGroupStatus(groupID uint, items map[uint]*monitorStatusItem, monitors map[uint]model.Monitor, children map[uint][]uint, visiting map[uint]bool) (uint16, float64) {
	status, uptimeSum, uptimeCount := aggregateGroupStatusStats(groupID, items, monitors, children, visiting)
	if uptimeCount == 0 {
		return status, 1.0
	}
	return status, uptimeSum / float64(uptimeCount)
}

func aggregateGroupStatusStats(groupID uint, items map[uint]*monitorStatusItem, monitors map[uint]model.Monitor, children map[uint][]uint, visiting map[uint]bool) (uint16, float64, int) {
	if visiting[groupID] {
		return model.StatusPending, 0, 0
	}
	visiting[groupID] = true
	defer delete(visiting, groupID)

	childIDs := children[groupID]
	if len(childIDs) == 0 {
		return model.StatusPending, 0, 0
	}

	status := model.StatusUP
	uptimeSum := 0.0
	uptimeCount := 0
	hasPending := false
	for _, childID := range childIDs {
		item, ok := items[childID]
		if !ok {
			continue
		}
		childStatus := item.Status
		if monitors[childID].Type == model.MonitorTypeGroup {
			var childUptimeSum float64
			var childUptimeCount int
			childStatus, childUptimeSum, childUptimeCount = aggregateGroupStatusStats(childID, items, monitors, children, visiting)
			uptimeSum += childUptimeSum
			uptimeCount += childUptimeCount
		} else {
			uptimeSum += item.Uptime24H
			uptimeCount++
		}
		if childStatus == model.StatusDown {
			status = model.StatusDown
		} else if childStatus == model.StatusPending {
			hasPending = true
		}
	}
	if status != model.StatusDown && hasPending {
		status = model.StatusPending
	}
	if uptimeCount == 0 {
		status = model.StatusPending
	}
	return status, uptimeSum, uptimeCount
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
