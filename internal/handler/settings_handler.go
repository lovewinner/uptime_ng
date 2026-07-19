package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

const SettingRegistrationOpen = "registration_open"

type SettingsHandler struct {
	DB *gorm.DB
}

func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{DB: db}
}

// GetSettings returns all settings (admin only).
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	var settings []model.Setting
	if err := h.DB.Find(&settings).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "settings_query_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, settings)
}

// UpdateSetting updates a single setting by key (admin only).
func (h *SettingsHandler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		badRequest(c, "missing_key", "setting key is required")
		return
	}

	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	setting := model.Setting{Key: key, Value: body.Value, Type: "string"}
	if err := h.DB.Where("key = ?", key).Assign(setting).FirstOrCreate(&setting).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "setting_update_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, setting)
}

// GetRegistrationStatus returns whether public registration is open (no auth required).
func (h *SettingsHandler) GetRegistrationStatus(c *gin.Context) {
	var setting model.Setting
	err := h.DB.Where("key = ?", SettingRegistrationOpen).First(&setting).Error
	open := err == nil && setting.Value == "true"
	c.JSON(http.StatusOK, gin.H{"registration_open": open})
}

// IsRegistrationOpen checks the DB for the registration toggle.
func IsRegistrationOpen(db *gorm.DB) bool {
	var setting model.Setting
	err := db.Where("key = ?", SettingRegistrationOpen).First(&setting).Error
	return err == nil && setting.Value == "true"
}
