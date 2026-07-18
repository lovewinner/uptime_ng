package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
	"uptime_ng/internal/notifier"
)

type NotificationHandler struct {
	DB *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{DB: db}
}

type CreateNotificationRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Config string `json:"config" binding:"required"` // JSON map
}

func (h *NotificationHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	notif, validationErr := notificationFromRequest(req, userID)
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	if err := h.DB.Create(&notif).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_create_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, notif)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	var notifs []model.Notification
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifs)

	c.JSON(http.StatusOK, notifs)
}

func (h *NotificationHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_notification_id", "invalid notification id")
		return
	}

	notif, err := userNotification(h.DB, userID, id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_notification_id", "invalid notification id")
		return
	}

	notif, err := userNotification(h.DB, userID, id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	if validationErr := applyNotificationRequest(&notif, req); validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	if err := h.DB.Save(&notif).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_notification_id", "invalid notification id")
		return
	}

	notif, err := userNotification(h.DB, userID, id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}
	if err := h.DB.Delete(&notif).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_delete_failed", err.Error())
		return
	}

	if err := h.DB.Where("notification_id = ?", id).Delete(&model.MonitorNotification{}).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_association_delete_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
}

func (h *NotificationHandler) Test(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_notification_id", "invalid notification id")
		return
	}

	notif, err := userNotification(h.DB, userID, id)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	configMap, err := notifier.ParseNotificationConfig(notif.Config)
	if err != nil {
		badRequest(c, "invalid_notification_config", "invalid notification config json")
		return
	}

	target, validationErr := notificationTestTargetFromConfig(notif, configMap)
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	switch notif.Type {
	case model.NotificationTypeFeishu:
		if err := notifier.NewFeishuNotifier(target.webhookURL, h.DB).SendText(target.message); err != nil {
			errorResponse(c, http.StatusBadGateway, "notification_send_failed", err.Error())
			return
		}
	case model.NotificationTypeEmail:
		n := notifier.NewEmailNotifierFromConfig(target.emailTo)
		if err := n.Send(target.emailSubject, target.emailBody); err != nil {
			errorResponse(c, http.StatusBadGateway, "notification_send_failed", err.Error())
			return
		}
	default:
		badRequest(c, "unsupported_notification_type", "unsupported notification type")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "测试消息已发送"})
}
