package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func monitorResponse(db *gorm.DB, monitor model.Monitor) (gin.H, error) {
	tags, err := monitorTags(db, monitor.ID)
	if err != nil {
		return nil, err
	}
	notificationIDs, err := monitorNotificationIDs(db, monitor.ID)
	if err != nil {
		return nil, err
	}
	return gin.H{
		"monitor":          monitor,
		"tags":             tags,
		"notification_ids": notificationIDs,
	}, nil
}

func monitorTags(db *gorm.DB, monitorID uint) ([]model.Tag, error) {
	var tags []model.Tag
	if err := db.Raw(`
		SELECT t.* FROM tags t
		JOIN monitor_tags mt ON mt.tag_id = t.id
		WHERE mt.monitor_id = ?
	`, monitorID).Scan(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func monitorNotificationIDs(db *gorm.DB, monitorID uint) ([]uint, error) {
	var notifs []model.MonitorNotification
	if err := db.Where("monitor_id = ?", monitorID).Find(&notifs).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(notifs))
	for i, notif := range notifs {
		ids[i] = notif.NotificationID
	}
	return ids, nil
}
