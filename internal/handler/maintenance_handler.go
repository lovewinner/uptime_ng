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
	if err := h.DB.Where("user_id = ?", userID).Order("start_at DESC").Find(&windows).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_list_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, windows)
}

func (h *MaintenanceHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req MaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	window, validationErr, lookupErr := h.buildMaintenanceWindow(req, userID, nil)
	if lookupErr != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_validation_failed", lookupErr.Error())
		return
	}
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
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
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_maintenance_id", "invalid maintenance id")
		return
	}
	existing, err := userMaintenanceWindow(h.DB, userID, id)
	if err != nil {
		lookupErrorResponse(c, err, "maintenance_not_found", "maintenance window not found", "maintenance_lookup_failed")
		return
	}
	var req MaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	window, validationErr, lookupErr := h.buildMaintenanceWindow(req, userID, &existing)
	if lookupErr != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_validation_failed", lookupErr.Error())
		return
	}
	if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
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
	id, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_maintenance_id", "invalid maintenance id")
		return
	}
	window, err := userMaintenanceWindow(h.DB, userID, id)
	if err != nil {
		lookupErrorResponse(c, err, "maintenance_not_found", "maintenance window not found", "maintenance_lookup_failed")
		return
	}
	if err := h.DB.Delete(&window).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "maintenance_delete_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "maintenance window deleted"})
}

func (h *MaintenanceHandler) buildMaintenanceWindow(req MaintenanceRequest, userID uint, existing *model.MaintenanceWindow) (model.MaintenanceWindow, *requestValidationError, error) {
	window, validationErr := maintenanceWindowFromRequest(req, userID, existing)
	if validationErr != nil {
		return model.MaintenanceWindow{}, validationErr, nil
	}
	if req.MonitorID != nil {
		if _, err := userMonitor(h.DB, userID, *req.MonitorID); err != nil {
			if isRecordNotFound(err) {
				return model.MaintenanceWindow{}, &requestValidationError{code: "invalid_monitor", message: "monitor_id must reference a monitor owned by the current user"}, nil
			}
			return model.MaintenanceWindow{}, nil, err
		}
	}

	return window, nil, nil
}
