package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type != "feishu" && req.Type != "email" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be feishu or email"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Update(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var notif model.Notification
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&notif).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notif.Name = req.Name
	notif.Type = req.Type
	notif.Config = req.Config

	if err := h.DB.Save(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	msg := fmt.Sprintf("🔔 来自 uptime_ng 的测试消息。通知: %s (%s)", notif.Name, notif.Type)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": msg, "note": "email/feishu sending will be implemented in P4"})
}