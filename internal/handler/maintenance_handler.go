package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type MaintenanceHandler struct {
	DB *gorm.DB
}

func NewMaintenanceHandler(db *gorm.DB) *MaintenanceHandler {
	return &MaintenanceHandler{DB: db}
}

type MaintenanceRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	MonitorID   *uint  `json:"monitor_id"`
	StartAt     string `json:"start_at" binding:"required"`
	EndAt       string `json:"end_at" binding:"required"`
	Active      *bool  `json:"active"`
}

func (h *MaintenanceHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	var windows []model.MaintenanceWindow
	h.DB.Where("user_id = ?", userID).Order("start_at DESC").Find(&windows)
	c.JSON(http.StatusOK, windows)
}

func (h *MaintenanceHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	window, ok := h.bindWindow(c, userID, nil)
	if !ok {
		return
	}
	if err := h.DB.Create(&window).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_create_failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, window)
}

func (h *MaintenanceHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var existing model.MaintenanceWindow
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "maintenance_not_found", "maintenance window not found")
		return
	}
	window, ok := h.bindWindow(c, userID, &existing)
	if !ok {
		return
	}
	if err := h.DB.Save(&window).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_update_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, window)
}

func (h *MaintenanceHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.MaintenanceWindow{}).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_delete_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "maintenance window deleted"})
}

func (h *MaintenanceHandler) bindWindow(c *gin.Context, userID uint, existing *model.MaintenanceWindow) (model.MaintenanceWindow, bool) {
	var req MaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return model.MaintenanceWindow{}, false
	}

	window, validationErr := maintenanceWindowFromRequest(req, userID, existing)
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return model.MaintenanceWindow{}, false
	}
	if req.MonitorID != nil {
		var monitor model.Monitor
		if err := h.DB.Where("id = ? AND user_id = ?", *req.MonitorID, userID).First(&monitor).Error; err != nil {
			badRequest(c, "invalid_monitor", "monitor_id must reference a monitor owned by the current user")
			return model.MaintenanceWindow{}, false
		}
	}

	return window, true
}
