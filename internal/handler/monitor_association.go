package handler

import (
	"errors"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

const defaultTagColor = "#409EFF"

func attachMonitorAssociations(tx *gorm.DB, monitorID uint, notificationIDs []uint, tagNames []string, tagColors []string) error {
	for _, nid := range notificationIDs {
		if err := tx.Create(&model.MonitorNotification{MonitorID: monitorID, NotificationID: nid}).Error; err != nil {
			return err
		}
	}

	for i, tagName := range tagNames {
		if tagName == "" {
			continue
		}
		tag, err := findOrCreateTag(tx, tagName, tagColorAt(tagColors, i))
		if err != nil {
			return err
		}
		if err := tx.Create(&model.MonitorTag{MonitorID: monitorID, TagID: tag.ID, Value: tagName}).Error; err != nil {
			return err
		}
	}
	return nil
}

func refreshMonitorAssociations(tx *gorm.DB, monitorID uint, notificationIDs []uint, tagNames []string, tagColors []string, refreshNotifications bool, refreshTags bool) error {
	if refreshNotifications {
		if err := tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorNotification{}).Error; err != nil {
			return err
		}
		for _, nid := range notificationIDs {
			if err := tx.Create(&model.MonitorNotification{MonitorID: monitorID, NotificationID: nid}).Error; err != nil {
				return err
			}
		}
	}

	if refreshTags {
		if err := tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorTag{}).Error; err != nil {
			return err
		}
		for i, tagName := range tagNames {
			if tagName == "" {
				continue
			}
			tag, err := findOrCreateTag(tx, tagName, tagColorAt(tagColors, i))
			if err != nil {
				return err
			}
			if err := tx.Create(&model.MonitorTag{MonitorID: monitorID, TagID: tag.ID, Value: tagName}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func attachExportMonitorAssociations(tx *gorm.DB, userID uint, monitorID uint, exported ExportMonitor) error {
	for _, et := range exported.Tags {
		if et.Name == "" {
			continue
		}
		tag, err := findOrCreateTag(tx, et.Name, tagColor(et.Color))
		if err != nil {
			return err
		}
		if err := tx.Create(&model.MonitorTag{MonitorID: monitorID, TagID: tag.ID, Value: et.Name}).Error; err != nil {
			return err
		}
	}

	for _, name := range exported.NotificationNames {
		var notification model.Notification
		err := tx.Where("user_id = ? AND name = ?", userID, name).First(&notification).Error
		if err == nil {
			if err := tx.Create(&model.MonitorNotification{MonitorID: monitorID, NotificationID: notification.ID}).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func refreshExportMonitorAssociations(tx *gorm.DB, userID uint, monitorID uint, exported ExportMonitor) error {
	if err := tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorTag{}).Error; err != nil {
		return err
	}
	if err := tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorNotification{}).Error; err != nil {
		return err
	}
	return attachExportMonitorAssociations(tx, userID, monitorID, exported)
}

func ungroupChildMonitors(tx *gorm.DB, groupID uint) error {
	return tx.Model(&model.Monitor{}).Where("group_id = ?", groupID).Update("group_id", nil).Error
}

func deleteMonitorData(tx *gorm.DB, monitor model.Monitor) error {
	monitorID := monitor.ID
	for _, modelValue := range []any{
		&model.Heartbeat{},
		&model.MonitorNotification{},
		&model.MonitorTag{},
		&model.MaintenanceWindow{},
		&model.StatMinutely{},
		&model.StatHourly{},
		&model.StatDaily{},
		&model.Incident{},
	} {
		if err := tx.Where("monitor_id = ?", monitorID).Delete(modelValue).Error; err != nil {
			return err
		}
	}
	if err := ungroupChildMonitors(tx, monitorID); err != nil {
		return err
	}
	return tx.Delete(&monitor).Error
}

func findOrCreateTag(tx *gorm.DB, name string, color string) (model.Tag, error) {
	var tag model.Tag
	err := tx.Where("name = ?", name).First(&tag).Error
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Tag{}, err
	}
	tag = model.Tag{Name: name, Color: color}
	if err := tx.Create(&tag).Error; err != nil {
		return model.Tag{}, err
	}
	return tag, nil
}

func tagColorAt(colors []string, index int) string {
	if index < len(colors) && colors[index] != "" {
		return colors[index]
	}
	return defaultTagColor
}

func tagColor(color string) string {
	if color != "" {
		return color
	}
	return defaultTagColor
}
