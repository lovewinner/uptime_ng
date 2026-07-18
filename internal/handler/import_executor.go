package handler

import (
	"errors"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type importMonitorAction string

const (
	importMonitorCreated importMonitorAction = "created"
	importMonitorUpdated importMonitorAction = "updated"
	importMonitorSkipped importMonitorAction = "skipped"
)

type importMonitorOutcome struct {
	action  importMonitorAction
	monitor model.Monitor
}

func importNotifications(tx *gorm.DB, userID uint, notifications []ExportNotification, strategy string) error {
	for _, en := range notifications {
		if en.Name == "" || en.Type == "" {
			continue
		}
		var existing model.Notification
		err := tx.Where("user_id = ? AND name = ?", userID, en.Name).First(&existing).Error
		if err == nil {
			if strategy == "overwrite" {
				existing.Type = en.Type
				if !containsMaskedValue(en.Config) {
					existing.Config = en.Config
				}
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if containsMaskedValue(en.Config) {
			continue
		}
		if err := tx.Create(&model.Notification{
			UserID: userID,
			Name:   en.Name,
			Type:   en.Type,
			Config: en.Config,
			Active: true,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func importMonitor(tx *gorm.DB, userID uint, exported ExportMonitor, strategy string) (importMonitorOutcome, error) {
	groupID, err := ensureGroupPath(tx, userID, exported.GroupPath)
	if err != nil {
		return importMonitorOutcome{}, err
	}

	var existing model.Monitor
	err = tx.Where("user_id = ? AND name = ?", userID, exported.Name).First(&existing).Error
	if err == nil {
		switch strategy {
		case "skip":
			return importMonitorOutcome{action: importMonitorSkipped}, nil
		case "overwrite":
			existing = applyExportMonitor(existing, exported)
			existing.GroupID = groupID
			if err := tx.Save(&existing).Error; err != nil {
				return importMonitorOutcome{}, err
			}
			if err := refreshExportMonitorAssociations(tx, userID, existing.ID, exported); err != nil {
				return importMonitorOutcome{}, err
			}
			return importMonitorOutcome{action: importMonitorUpdated, monitor: existing}, nil
		case "copy":
			exported.Name = exported.Name + " (copy)"
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return importMonitorOutcome{}, err
	}

	monitor := newMonitorFromExport(userID, exported, groupID)
	if err := tx.Create(&monitor).Error; err != nil {
		return importMonitorOutcome{}, err
	}
	if err := attachExportMonitorAssociations(tx, userID, monitor.ID, exported); err != nil {
		return importMonitorOutcome{}, err
	}
	return importMonitorOutcome{action: importMonitorCreated, monitor: monitor}, nil
}

func syncImportedMonitorSchedulers(scheduler MonitorScheduler, monitors []model.Monitor) {
	if scheduler == nil {
		return
	}
	for i := range monitors {
		if monitors[i].Type == model.MonitorTypeGroup {
			scheduler.StopMonitor(monitors[i].ID)
		} else if monitors[i].Active {
			scheduler.RestartMonitor(&monitors[i])
		} else {
			scheduler.StopMonitor(monitors[i].ID)
		}
	}
}
