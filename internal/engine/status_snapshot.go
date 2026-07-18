package engine

import "uptime_ng/internal/model"

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
