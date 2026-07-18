package handler

import (
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func userNotification(db *gorm.DB, userID uint, notificationID uint) (model.Notification, error) {
	var notification model.Notification
	err := db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error
	return notification, err
}

func userMaintenanceWindow(db *gorm.DB, userID uint, windowID uint) (model.MaintenanceWindow, error) {
	var window model.MaintenanceWindow
	err := db.Where("id = ? AND user_id = ?", windowID, userID).First(&window).Error
	return window, err
}
