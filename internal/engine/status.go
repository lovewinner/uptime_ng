package engine

import (
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type MonitorStatusSnapshot struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	GroupID   *uint   `json:"group_id"`
	Status    uint16  `json:"status"`
	PingMS    float64 `json:"ping_ms"`
	Uptime24H float64 `json:"uptime_24h"`
	Active    bool    `json:"active"`
}

func ComputeMonitorStatus(db *gorm.DB, userID uint, monitorID uint) (MonitorStatusSnapshot, error) {
	var monitor model.Monitor
	if err := db.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		return MonitorStatusSnapshot{}, err
	}
	return computeMonitorStatus(db, monitor, map[uint]bool{})
}

func ComputeActiveStatuses(db *gorm.DB, userID uint) ([]MonitorStatusSnapshot, error) {
	var monitors []model.Monitor
	if err := db.Where("user_id = ? AND active = ?", userID, true).Find(&monitors).Error; err != nil {
		return nil, err
	}

	results := make([]MonitorStatusSnapshot, len(monitors))
	for i, monitor := range monitors {
		status, err := computeMonitorStatus(db, monitor, map[uint]bool{})
		if err != nil {
			return nil, err
		}
		results[i] = status
	}
	return results, nil
}

func computeMonitorStatus(db *gorm.DB, monitor model.Monitor, visiting map[uint]bool) (MonitorStatusSnapshot, error) {
	item := MonitorStatusSnapshot{
		ID:        monitor.ID,
		Name:      monitor.Name,
		Type:      monitor.Type,
		GroupID:   monitor.GroupID,
		Status:    model.StatusPending,
		Uptime24H: 1.0,
		Active:    monitor.Active,
	}
	if monitor.Type == model.MonitorTypeGroup {
		status, uptime := computeGroupStatus(db, monitor, visiting)
		item.Status = status
		item.Uptime24H = uptime
		return item, nil
	}
	if !monitor.Active {
		return item, nil
	}

	item.Status = model.StatusDown
	var beat model.Heartbeat
	if db.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&beat).Error == nil {
		item.Status = beat.Status
		if beat.PingMS != nil {
			item.PingMS = *beat.PingMS
		}
	}
	item.Uptime24H = monitorUptime24H(db, monitor.ID)
	return item, nil
}

func computeGroupStatus(db *gorm.DB, group model.Monitor, visiting map[uint]bool) (uint16, float64) {
	if visiting[group.ID] {
		return model.StatusPending, 1.0
	}
	visiting[group.ID] = true
	defer delete(visiting, group.ID)

	var children []model.Monitor
	db.Where("user_id = ? AND group_id = ?", group.UserID, group.ID).Find(&children)
	if len(children) == 0 {
		return model.StatusPending, 1.0
	}

	status := model.StatusUP
	hasPending := false
	uptimeSum := 0.0
	uptimeCount := 0
	for _, child := range children {
		childStatus, err := computeMonitorStatus(db, child, visiting)
		if err != nil {
			hasPending = true
			continue
		}
		if childStatus.Status == model.StatusDown {
			status = model.StatusDown
		} else if childStatus.Status != model.StatusUP {
			hasPending = true
		}
		uptimeSum += childStatus.Uptime24H
		uptimeCount++
	}
	if status != model.StatusDown && hasPending {
		status = model.StatusPending
	}
	if uptimeCount == 0 {
		return model.StatusPending, 1.0
	}
	return status, uptimeSum / float64(uptimeCount)
}

func monitorUptime24H(db *gorm.DB, monitorID uint) float64 {
	var up, down int64
	cutoff := time.Now().Add(-24 * time.Hour)
	db.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusUP).Count(&up)
	db.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusDown).Count(&down)
	if up+down > 0 {
		return float64(up) / float64(up+down)
	}
	return 1.0
}

func GroupStatusMessage(status uint16) string {
	switch status {
	case model.StatusUP:
		return "group OK: all child monitors are UP"
	case model.StatusDown:
		return "group DOWN: at least one child monitor is DOWN"
	default:
		return "group PENDING: child monitors are pending or group is empty"
	}
}
