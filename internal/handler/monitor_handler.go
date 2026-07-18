package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type MonitorHandler struct {
	DB        *gorm.DB
	Scheduler MonitorScheduler
}

type MonitorScheduler interface {
	StartMonitor(monitor *model.Monitor)
	StopMonitor(monitorID uint)
	RestartMonitor(monitor *model.Monitor)
}

func NewMonitorHandler(db *gorm.DB, scheduler MonitorScheduler) *MonitorHandler {
	return &MonitorHandler{DB: db, Scheduler: scheduler}
}

type CreateMonitorRequest struct {
	Name                  string   `json:"name" binding:"required"`
	Description           string   `json:"description"`
	Type                  string   `json:"type" binding:"required"`
	GroupID               *uint    `json:"group_id"`
	URL                   string   `json:"url"`
	Hostname              string   `json:"hostname"`
	Port                  uint16   `json:"port"`
	Method                string   `json:"method"`
	Interval              uint32   `json:"interval"`
	Timeout               float64  `json:"timeout"`
	MaxRetries            uint32   `json:"max_retries"`
	RetryInterval         uint32   `json:"retry_interval"`
	ResendInterval        uint32   `json:"resend_interval"`
	Headers               string   `json:"headers"`
	Body                  string   `json:"body"`
	AcceptedStatusCodes   []string `json:"accepted_status_codes"`
	Keyword               string   `json:"keyword"`
	InvertKeyword         bool     `json:"invert_keyword"`
	IgnoreTLS             bool     `json:"ignore_tls"`
	UpsideDown            bool     `json:"upside_down"`
	MaxRedirects          uint32   `json:"max_redirects"`
	AuthMethod            string   `json:"auth_method"`
	BasicAuthUser         string   `json:"basic_auth_user"`
	BasicAuthPass         string   `json:"basic_auth_pass"`
	BearerToken           string   `json:"bearer_token"`
	AuthWorkstation       string   `json:"auth_workstation"`
	AuthDomain            string   `json:"auth_domain"`
	TLSKey                string   `json:"tls_key"`
	TLSCert               string   `json:"tls_cert"`
	TLSCa                 string   `json:"tls_ca"`
	OAuthClientID         string   `json:"oauth_client_id"`
	OAuthClientSecret     string   `json:"oauth_client_secret"`
	OAuthTokenURL         string   `json:"oauth_token_url"`
	OAuthScopes           string   `json:"oauth_scopes"`
	OAuthAuthMethod       string   `json:"oauth_auth_method"`
	OAuthAudience         string   `json:"oauth_audience"`
	DNSResolveType        string   `json:"dns_resolve_type"`
	DNSResolveServer      string   `json:"dns_resolve_server"`
	PacketSize            uint32   `json:"packet_size"`
	ExpiryNotification    *bool    `json:"expiry_notification"`
	HTTPBodyEncoding      string   `json:"http_body_encoding"`
	RetryOnlyOnStatusCode bool     `json:"retry_only_on_status_code"`
	CacheBust             bool     `json:"cache_bust"`
	SaveResponse          bool     `json:"save_response"`
	SaveErrorResponse     bool     `json:"save_error_response"`
	ResponseMaxLength     uint32   `json:"response_max_length"`
	PingNumeric           bool     `json:"ping_numeric"`
	PingCount             uint32   `json:"ping_count"`
	PingPerRequestTimeout uint32   `json:"ping_per_request_timeout"`
	NotificationIDs       []uint   `json:"notification_ids"`
	TagNames              []string `json:"tag_names"`
	TagColors             []string `json:"tag_colors"`
}

func (h *MonitorHandler) Create(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	normalizeMonitorRequest(&req)
	if err := h.validateMonitorRequest(userID.(uint), 0, req); err != nil {
		badRequest(c, err.code, err.message)
		return
	}

	monitor := newMonitorFromRequest(userID.(uint), req)

	tx := h.DB.Begin()

	if err := tx.Create(&monitor).Error; err != nil {
		tx.Rollback()
		errorResponse(c, http.StatusInternalServerError, "monitor_create_failed", err.Error())
		return
	}

	attachMonitorAssociations(tx, monitor.ID, req.NotificationIDs, req.TagNames, req.TagColors)

	if err := tx.Commit().Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_create_commit_failed", err.Error())
		return
	}

	if monitor.Active && h.Scheduler != nil {
		h.Scheduler.StartMonitor(&monitor)
	}

	c.JSON(http.StatusCreated, monitor)
}

func (h *MonitorHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var monitors []model.Monitor
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&monitors)

	results := make([]gin.H, len(monitors))

	for i, m := range monitors {
		results[i] = monitorResponse(h.DB, m)
	}

	c.JSON(http.StatusOK, results)
}

func (h *MonitorHandler) Get(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	c.JSON(http.StatusOK, monitorResponse(h.DB, monitor))
}

func (h *MonitorHandler) Update(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	normalizeMonitorRequest(&req)
	if err := h.validateMonitorRequest(userID.(uint), monitor.ID, req); err != nil {
		badRequest(c, err.code, err.message)
		return
	}

	wasGroup := monitor.Type == model.MonitorTypeGroup
	applyMonitorRequest(&monitor, req, false)

	tx := h.DB.Begin()
	if wasGroup && monitor.Type != model.MonitorTypeGroup {
		tx.Model(&model.Monitor{}).Where("group_id = ?", monitor.ID).Update("group_id", nil)
	}
	if err := tx.Save(&monitor).Error; err != nil {
		tx.Rollback()
		errorResponse(c, http.StatusInternalServerError, "monitor_update_failed", err.Error())
		return
	}

	refreshMonitorAssociations(tx, monitor.ID, req.NotificationIDs, req.TagNames, req.TagColors, req.NotificationIDs != nil, req.TagNames != nil)

	if err := tx.Commit().Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_update_commit_failed", err.Error())
		return
	}

	if h.Scheduler != nil {
		if monitor.Active {
			h.Scheduler.RestartMonitor(&monitor)
		} else {
			h.Scheduler.StopMonitor(monitor.ID)
		}
	}

	c.JSON(http.StatusOK, monitor)
}

func (h *MonitorHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}

	tx := h.DB.Begin()
	tx.Where("monitor_id = ?", monitorID).Delete(&model.Heartbeat{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorNotification{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorTag{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.MaintenanceWindow{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatMinutely{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatHourly{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatDaily{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.Incident{})
	tx.Model(&model.Monitor{}).Where("group_id = ?", monitor.ID).Update("group_id", nil)
	tx.Delete(&monitor)
	if err := tx.Commit().Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_delete_failed", err.Error())
		return
	}

	if h.Scheduler != nil {
		h.Scheduler.StopMonitor(monitor.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "monitor deleted"})
}

func (h *MonitorHandler) Resume(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}
	monitors := h.monitorActivationTargets(userID.(uint), monitor)
	ids := monitorIDs(monitors)
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", true).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_resume_failed", err.Error())
		return
	}
	restartMonitors(h.Scheduler, monitors)
	c.JSON(http.StatusOK, gin.H{"message": "monitor resumed"})
}

func (h *MonitorHandler) Pause(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		errorResponse(c, http.StatusNotFound, "monitor_not_found", "monitor not found")
		return
	}
	monitors := h.monitorActivationTargets(userID.(uint), monitor)
	ids := monitorIDs(monitors)
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", false).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_pause_failed", err.Error())
		return
	}
	stopMonitors(h.Scheduler, monitors)
	c.JSON(http.StatusOK, gin.H{"message": "monitor paused"})
}

func (h *MonitorHandler) validateMonitorRequest(userID uint, monitorID uint, req CreateMonitorRequest) *requestValidationError {
	if !isValidMonitorType(req.Type) {
		return &requestValidationError{code: "invalid_monitor_type", message: "type must be http, tcp, ping, dns, push or group"}
	}
	if req.GroupID == nil {
		return nil
	}
	if monitorID != 0 && *req.GroupID == monitorID {
		return &requestValidationError{code: "invalid_group", message: "monitor cannot be its own group"}
	}

	var parent model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ? AND type = ?", *req.GroupID, userID, model.MonitorTypeGroup).First(&parent).Error; err != nil {
		return &requestValidationError{code: "invalid_group", message: "group_id must reference a group monitor owned by the current user"}
	}
	if monitorID != 0 && h.wouldCreateGroupCycle(userID, monitorID, parent.ID) {
		return &requestValidationError{code: "group_cycle", message: "group hierarchy cannot contain cycles"}
	}
	return nil
}

func isValidMonitorType(monitorType string) bool {
	switch monitorType {
	case model.MonitorTypeHTTP, model.MonitorTypeTCP, model.MonitorTypePing, model.MonitorTypeDNS, model.MonitorTypePush, model.MonitorTypeGroup:
		return true
	default:
		return false
	}
}

func (h *MonitorHandler) wouldCreateGroupCycle(userID uint, monitorID uint, parentID uint) bool {
	seen := map[uint]bool{}
	current := parentID
	for current != 0 {
		if current == monitorID {
			return true
		}
		if seen[current] {
			return true
		}
		seen[current] = true

		var parent model.Monitor
		if err := h.DB.Select("id", "group_id").Where("id = ? AND user_id = ?", current, userID).First(&parent).Error; err != nil {
			return false
		}
		if parent.GroupID == nil {
			return false
		}
		current = *parent.GroupID
	}
	return false
}

func (h *MonitorHandler) descendantMonitors(userID uint, groupID uint) []model.Monitor {
	var children []model.Monitor
	h.DB.Where("user_id = ? AND group_id = ?", userID, groupID).Find(&children)
	results := make([]model.Monitor, 0, len(children))
	for _, child := range children {
		results = append(results, child)
		if child.Type == model.MonitorTypeGroup {
			results = append(results, h.descendantMonitors(userID, child.ID)...)
		}
	}
	return results
}
