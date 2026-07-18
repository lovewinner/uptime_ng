package handler

import (
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func importNotifications(tx *gorm.DB, userID uint, notifications []ExportNotification, strategy string) {
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
				tx.Save(&existing)
			}
			continue
		}
		if containsMaskedValue(en.Config) {
			continue
		}
		tx.Create(&model.Notification{
			UserID: userID,
			Name:   en.Name,
			Type:   en.Type,
			Config: en.Config,
			Active: true,
		})
	}
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
