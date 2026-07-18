package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func monitorResponse(db *gorm.DB, monitor model.Monitor) gin.H {
	return gin.H{
		"monitor":          monitor,
		"tags":             monitorTags(db, monitor.ID),
		"notification_ids": monitorNotificationIDs(db, monitor.ID),
	}
}

func monitorTags(db *gorm.DB, monitorID uint) []model.Tag {
	var tags []model.Tag
	db.Raw(`
		SELECT t.* FROM tags t
		JOIN monitor_tags mt ON mt.tag_id = t.id
		WHERE mt.monitor_id = ?
	`, monitorID).Scan(&tags)
	return tags
}

func monitorNotificationIDs(db *gorm.DB, monitorID uint) []uint {
	var notifs []model.MonitorNotification
	db.Where("monitor_id = ?", monitorID).Find(&notifs)
	ids := make([]uint, len(notifs))
	for i, notif := range notifs {
		ids[i] = notif.NotificationID
	}
	return ids
}
