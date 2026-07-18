package handler

import (
	"errors"

	"uptime_ng/internal/model"
)

func (h *MonitorHandler) monitorActivationTargets(userID uint, root model.Monitor) ([]model.Monitor, error) {
	if root.Type != model.MonitorTypeGroup {
		return []model.Monitor{root}, nil
	}
	descendants, err := h.descendantMonitors(userID, root.ID)
	if err != nil {
		return nil, err
	}
	return monitorActivationTargets(root, descendants), nil
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

func restartMonitors(scheduler MonitorScheduler, monitors []model.Monitor) error {
	if scheduler == nil {
		return nil
	}
	var errs []error
	for i := range monitors {
		monitors[i].Active = true
		if err := scheduler.RestartMonitor(&monitors[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func stopMonitors(scheduler MonitorScheduler, monitors []model.Monitor) {
	if scheduler == nil {
		return
	}
	for _, monitor := range monitors {
		scheduler.StopMonitor(monitor.ID)
	}
}
