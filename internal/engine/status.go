package engine

import (
	"errors"
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

func pendingStatusSnapshot(monitor model.Monitor) MonitorStatusSnapshot {
	return MonitorStatusSnapshot{
		ID:        monitor.ID,
		Name:      monitor.Name,
		Type:      monitor.Type,
		GroupID:   monitor.GroupID,
		Status:    model.StatusPending,
		Uptime24H: 1.0,
		Active:    monitor.Active,
	}
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
	item := pendingStatusSnapshot(monitor)
	if monitor.Type == model.MonitorTypeGroup {
		status, uptime, err := computeGroupStatus(db, monitor, visiting)
		if err != nil {
			return MonitorStatusSnapshot{}, err
		}
		item.Status = status
		item.Uptime24H = uptime
		return item, nil
	}
	if !monitor.Active {
		return item, nil
	}

	item.Status = model.StatusDown
	var beat model.Heartbeat
	if err := db.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&beat).Error; err == nil {
		item.Status = beat.Status
		if beat.PingMS != nil {
			item.PingMS = *beat.PingMS
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return MonitorStatusSnapshot{}, err
	}
	uptime, err := monitorUptime24H(db, monitor.ID)
	if err != nil {
		return MonitorStatusSnapshot{}, err
	}
	item.Uptime24H = uptime
	return item, nil
}

func computeGroupStatus(db *gorm.DB, group model.Monitor, visiting map[uint]bool) (uint16, float64, error) {
	if visiting[group.ID] {
		return model.StatusPending, 1.0, nil
	}
	visiting[group.ID] = true
	defer delete(visiting, group.ID)

	var children []model.Monitor
	if err := db.Where("user_id = ? AND group_id = ?", group.UserID, group.ID).Find(&children).Error; err != nil {
		return model.StatusPending, 1.0, err
	}
	if len(children) == 0 {
		return model.StatusPending, 1.0, nil
	}

	acc := groupStatusAccumulator{}
	for _, child := range children {
		childStatus, err := computeMonitorStatus(db, child, visiting)
		if err != nil {
			acc.addPending()
			continue
		}
		acc.addChild(childStatus)
	}
	status, uptime := acc.result(); return status, uptime, nil
}

func monitorUptime24H(db *gorm.DB, monitorID uint) (float64, error) {
	var up, down int64
	cutoff := time.Now().Add(-24 * time.Hour)
	if err := db.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusUP).Count(&up).Error; err != nil {
		return 1.0, err
	}
	if err := db.Model(&model.Heartbeat{}).Where("monitor_id = ? AND time > ? AND status = ?", monitorID, cutoff, model.StatusDown).Count(&down).Error; err != nil {
		return 1.0, err
	}
	if up+down > 0 {
		return float64(up) / float64(up+down), nil
	}
	return 1.0, nil
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

// --- group status accumulator (moved from group_status.go) ---

type groupStatusAccumulator struct {
	hasDown    bool
	hasPending bool
	uptimeSum  float64
	uptimeNum  int
}

func (a *groupStatusAccumulator) addChild(child MonitorStatusSnapshot) {
	if child.Status == model.StatusDown {
		a.hasDown = true
	} else if child.Status != model.StatusUP {
		a.hasPending = true
	}
	a.uptimeSum += child.Uptime24H
	a.uptimeNum++
}

func (a *groupStatusAccumulator) addPending() {
	a.hasPending = true
}

func (a groupStatusAccumulator) result() (uint16, float64) {
	if a.uptimeNum == 0 {
		return model.StatusPending, 1.0
	}
	status := model.StatusUP
	if a.hasDown {
		status = model.StatusDown
	} else if a.hasPending {
		status = model.StatusPending
	}
	return status, a.uptimeSum / float64(a.uptimeNum)
}
