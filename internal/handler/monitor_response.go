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
	notificationIDs, err := userMonitorNotificationIDs(db, monitor.UserID, monitor.ID)
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
	err := db.Model(&model.Tag{}).
		Select("tags.*").
		Joins("JOIN monitor_tags mt ON mt.tag_id = tags.id").
		Where("mt.monitor_id = ?", monitorID).
		Find(&tags).Error
	return tags, err
}

func userMonitorNotifications(db *gorm.DB, userID uint, monitorID uint) ([]model.Notification, error) {
	var notifications []model.Notification
	err := db.Model(&model.Notification{}).
		Select("notifications.*").
		Joins("JOIN monitor_notifications mn ON mn.notification_id = notifications.id").
		Where("notifications.user_id = ? AND mn.monitor_id = ?", userID, monitorID).
		Find(&notifications).Error
	return notifications, err
}

func userMonitorNotificationIDs(db *gorm.DB, userID uint, monitorID uint) ([]uint, error) {
	var ids []uint
	if err := db.Model(&model.Notification{}).
		Select("notifications.id").
		Joins("JOIN monitor_notifications mn ON mn.notification_id = notifications.id").
		Where("notifications.user_id = ? AND mn.monitor_id = ?", userID, monitorID).
		Pluck("notifications.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
