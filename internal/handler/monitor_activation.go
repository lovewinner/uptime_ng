package handler

import (
	"errors"

	"gorm.io/gorm"

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

func setMonitorActivation(db *gorm.DB, scheduler MonitorScheduler, monitors []model.Monitor, active bool) error {
	ids := monitorIDs(monitors)
	if err := db.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", active).Error; err != nil {
		return err
	}
	if !active {
		stopMonitors(scheduler, monitors)
		return nil
	}
	if err := restartMonitors(scheduler, monitors); err != nil {
		stopMonitors(scheduler, monitors)
		rollbackErr := restoreMonitorActivation(db, scheduler, monitors)
		return errors.Join(err, rollbackErr)
	}
	return nil
}

func rollbackCreatedMonitor(db *gorm.DB, scheduler MonitorScheduler, monitor model.Monitor, cause error) error {
	if scheduler != nil {
		scheduler.StopMonitor(monitor.ID)
	}
	cleanupErr := runTransaction(db, func(tx *gorm.DB) error {
		return deleteMonitorData(tx, monitor)
	})
	return errors.Join(cause, cleanupErr)
}

func deactivateMonitorAfterSchedulerFailure(db *gorm.DB, scheduler MonitorScheduler, monitor *model.Monitor, cause error) error {
	if scheduler != nil {
		scheduler.StopMonitor(monitor.ID)
	}
	monitor.Active = false
	updateErr := db.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("active", false).Error
	return errors.Join(cause, updateErr)
}

func restoreMonitorActivation(db *gorm.DB, scheduler MonitorScheduler, monitors []model.Monitor) error {
	var errs []error
	for _, monitor := range monitors {
		if err := db.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("active", monitor.Active).Error; err != nil {
			errs = append(errs, err)
			continue
		}
		if scheduler != nil && shouldRunScheduledMonitor(monitor) {
			restored := monitor
			if err := scheduler.RestartMonitor(&restored); err != nil {
				errs = append(errs, deactivateMonitorAfterSchedulerFailure(db, scheduler, &restored, err))
			}
		}
	}
	return errors.Join(errs...)
}

func restartMonitors(scheduler MonitorScheduler, monitors []model.Monitor) error {
	if scheduler == nil {
		return nil
	}
	var errs []error
	for i := range monitors {
		monitor := monitors[i]
		monitor.Active = true
		if !shouldRunScheduledMonitor(monitor) {
			scheduler.StopMonitor(monitor.ID)
			continue
		}
		if err := scheduler.RestartMonitor(&monitor); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func shouldRunScheduledMonitor(monitor model.Monitor) bool {
	return monitor.Active && monitor.Type != model.MonitorTypePush
}

func stopMonitors(scheduler MonitorScheduler, monitors []model.Monitor) {
	if scheduler == nil {
		return
	}
	for _, monitor := range monitors {
		scheduler.StopMonitor(monitor.ID)
	}
}
