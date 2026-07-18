package handler

import (
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

const defaultTagColor = "#409EFF"

func attachMonitorAssociations(tx *gorm.DB, monitorID uint, notificationIDs []uint, tagNames []string, tagColors []string) {
	for _, nid := range notificationIDs {
		tx.Create(&model.MonitorNotification{MonitorID: monitorID, NotificationID: nid})
	}

	for i, tagName := range tagNames {
		if tagName == "" {
			continue
		}
		tag := findOrCreateTag(tx, tagName, tagColorAt(tagColors, i))
		tx.Create(&model.MonitorTag{MonitorID: monitorID, TagID: tag.ID, Value: tagName})
	}
}

func refreshMonitorAssociations(tx *gorm.DB, monitorID uint, notificationIDs []uint, tagNames []string, tagColors []string, refreshNotifications bool, refreshTags bool) {
	if refreshNotifications {
		tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorNotification{})
		for _, nid := range notificationIDs {
			tx.Create(&model.MonitorNotification{MonitorID: monitorID, NotificationID: nid})
		}
	}

	if refreshTags {
		tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorTag{})
		for i, tagName := range tagNames {
			if tagName == "" {
				continue
			}
			tag := findOrCreateTag(tx, tagName, tagColorAt(tagColors, i))
			tx.Create(&model.MonitorTag{MonitorID: monitorID, TagID: tag.ID, Value: tagName})
		}
	}
}

func findOrCreateTag(tx *gorm.DB, name string, color string) model.Tag {
	var tag model.Tag
	if err := tx.Where("name = ?", name).First(&tag).Error; err != nil {
		tag = model.Tag{Name: name, Color: color}
		tx.Create(&tag)
	}
	return tag
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
