package handler

import (
	"encoding/json"
	"fmt"
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
	userID, _ := c.Get("user_id")

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	if req.Type != "feishu" && req.Type != "email" {
		badRequest(c, "invalid_notification_type", "type must be feishu or email")
		return
	}

	notif := model.Notification{
		UserID: userID.(uint),
		Name:   req.Name,
		Type:   req.Type,
		Config: req.Config,
		Active: true,
	}

	if err := h.DB.Create(&notif).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_create_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, notif)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var notifs []model.Notification
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifs)

	c.JSON(http.StatusOK, notifs)
}

func (h *NotificationHandler) Get(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var notif model.Notification
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&notif).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Update(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var notif model.Notification
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&notif).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	if req.Type != "feishu" && req.Type != "email" {
		badRequest(c, "invalid_notification_type", "type must be feishu or email")
		return
	}

	notif.Name = req.Name
	notif.Type = req.Type
	notif.Config = req.Config

	if err := h.DB.Save(&notif).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Notification{}).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "notification_delete_failed", err.Error())
		return
	}

	h.DB.Where("notification_id = ?", id).Delete(&model.MonitorNotification{})

	c.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
}

func (h *NotificationHandler) Test(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var notif model.Notification
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&notif).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "notification_not_found", "notification not found")
		return
	}

	var configMap map[string]string
	if err := json.Unmarshal([]byte(notif.Config), &configMap); err != nil {
		badRequest(c, "invalid_notification_config", "invalid notification config json")
		return
	}

	msg := fmt.Sprintf("来自 uptime_ng 的测试消息。通知: %s (%s)", notif.Name, notif.Type)
	switch notif.Type {
	case "feishu":
		webhookURL := configMap["webhook_url"]
		if webhookURL == "" {
			badRequest(c, "missing_webhook_url", "missing webhook_url")
			return
		}
		if err := notifier.NewFeishuNotifier(webhookURL, h.DB).SendText(msg); err != nil {
			errorResponse(c, http.StatusBadGateway, "notification_send_failed", err.Error())
			return
		}
	case "email":
		to := configMap["email"]
		if to == "" {
			to = configMap["to"]
		}
		if cc := configMap["cc"]; cc != "" {
			to += "," + cc
		}
		if to == "" {
			badRequest(c, "missing_email_recipient", "missing email recipient")
			return
		}
		n := notifier.NewEmailNotifierFromConfig(to)
		if err := n.Send("[uptime_ng] 通知测试", "<p>"+msg+"</p>"); err != nil {
			errorResponse(c, http.StatusBadGateway, "notification_send_failed", err.Error())
			return
		}
	default:
		badRequest(c, "unsupported_notification_type", "unsupported notification type")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "测试消息已发送"})
}
