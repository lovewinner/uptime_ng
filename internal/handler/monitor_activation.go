package handler

import "uptime_ng/internal/model"

func (h *MonitorHandler) monitorActivationTargets(userID uint, root model.Monitor) []model.Monitor {
	if root.Type != model.MonitorTypeGroup {
		return []model.Monitor{root}
	}
	return monitorActivationTargets(root, h.descendantMonitors(userID, root.ID))
}

func monitorActivationTargets(root model.Monitor, descendants []model.Monitor) []model.Monitor {
	if root.Type != model.MonitorTypeGroup {
		return []model.Monitor{root}
	}
	monitors := make([]model.Monitor, 0, 1+len(descendants))
	monitors = append(monitors, root)
	monitors = append(monitors, descendants...)
	return monitors
}

func monitorIDs(monitors []model.Monitor) []uint {
	ids := make([]uint, 0, len(monitors))
	for _, monitor := range monitors {
		ids = append(ids, monitor.ID)
	}
	return ids
}

func restartMonitors(scheduler MonitorScheduler, monitors []model.Monitor) {
	if scheduler == nil {
		return
	}
	for i := range monitors {
		monitors[i].Active = true
		scheduler.RestartMonitor(&monitors[i])
	}
}

func stopMonitors(scheduler MonitorScheduler, monitors []model.Monitor) {
	if scheduler == nil {
		return
	}
	for _, monitor := range monitors {
		scheduler.StopMonitor(monitor.ID)
	}
}
