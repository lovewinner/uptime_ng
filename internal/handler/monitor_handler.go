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
	StartMonitor(monitor *model.Monitor) error
	StopMonitor(monitorID uint)
	RestartMonitor(monitor *model.Monitor) error
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
	userID := c.GetUint("user_id")

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	normalizeMonitorRequest(&req)
	if validationErr, lookupErr := h.validateMonitorRequest(userID, 0, req); lookupErr != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_validation_failed", lookupErr.Error())
		return
	} else if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	monitor := newMonitorFromRequest(userID, req)

	if err := runTransaction(h.DB, func(tx *gorm.DB) error {
		if err := tx.Create(&monitor).Error; err != nil {
			return err
		}
		return attachMonitorAssociations(tx, monitor.ID, req.NotificationIDs, req.TagNames, req.TagColors)
	}); err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_create_failed", err.Error())
		return
	}

	if monitor.Active && h.Scheduler != nil {
		if err := h.Scheduler.StartMonitor(&monitor); err != nil {
			errorResponse(c, http.StatusInternalServerError, "monitor_scheduler_start_failed", err.Error())
			return
		}
	}

	c.JSON(http.StatusCreated, monitor)
}

func (h *MonitorHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	var monitors []model.Monitor
	if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&monitors).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_list_failed", err.Error())
		return
	}

	results := make([]gin.H, len(monitors))

	for i, m := range monitors {
		response, err := monitorResponse(h.DB, m)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "monitor_response_failed", err.Error())
			return
		}
		results[i] = response
	}

	c.JSON(http.StatusOK, results)
}

func (h *MonitorHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	monitor, err := userMonitor(h.DB, userID, monitorID)
	if err != nil {
		lookupErrorResponse(c, err, "monitor_not_found", "monitor not found", "monitor_lookup_failed")
		return
	}

	response, err := monitorResponse(h.DB, monitor)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_response_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *MonitorHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	monitor, err := userMonitor(h.DB, userID, monitorID)
	if err != nil {
		lookupErrorResponse(c, err, "monitor_not_found", "monitor not found", "monitor_lookup_failed")
		return
	}

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}
	normalizeMonitorRequest(&req)
	if validationErr, lookupErr := h.validateMonitorRequest(userID, monitor.ID, req); lookupErr != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_validation_failed", lookupErr.Error())
		return
	} else if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	wasGroup := monitor.Type == model.MonitorTypeGroup
	applyMonitorRequest(&monitor, req, false)

	if err := runTransaction(h.DB, func(tx *gorm.DB) error {
		if wasGroup && monitor.Type != model.MonitorTypeGroup {
			if err := ungroupChildMonitors(tx, monitor.ID); err != nil {
				return err
			}
		}
		if err := tx.Save(&monitor).Error; err != nil {
			return err
		}
		return refreshMonitorAssociations(tx, monitor.ID, req.NotificationIDs, req.TagNames, req.TagColors, req.NotificationIDs != nil, req.TagNames != nil)
	}); err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_update_failed", err.Error())
		return
	}

	if h.Scheduler != nil {
		if monitor.Active {
			if err := h.Scheduler.RestartMonitor(&monitor); err != nil {
				errorResponse(c, http.StatusInternalServerError, "monitor_scheduler_restart_failed", err.Error())
				return
			}
		} else {
			h.Scheduler.StopMonitor(monitor.ID)
		}
	}

	c.JSON(http.StatusOK, monitor)
}

func (h *MonitorHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	monitor, err := userMonitor(h.DB, userID, monitorID)
	if err != nil {
		lookupErrorResponse(c, err, "monitor_not_found", "monitor not found", "monitor_lookup_failed")
		return
	}

	if err := runTransaction(h.DB, func(tx *gorm.DB) error {
		return deleteMonitorData(tx, monitor)
	}); err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_delete_failed", err.Error())
		return
	}

	if h.Scheduler != nil {
		h.Scheduler.StopMonitor(monitor.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "monitor deleted"})
}

func (h *MonitorHandler) Resume(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	monitor, err := userMonitor(h.DB, userID, monitorID)
	if err != nil {
		lookupErrorResponse(c, err, "monitor_not_found", "monitor not found", "monitor_lookup_failed")
		return
	}
	monitors, err := h.monitorActivationTargets(userID, monitor)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_descendants_failed", err.Error())
		return
	}
	ids := monitorIDs(monitors)
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", true).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_resume_failed", err.Error())
		return
	}
	if err := restartMonitors(h.Scheduler, monitors); err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_scheduler_restart_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "monitor resumed"})
}

func (h *MonitorHandler) Pause(c *gin.Context) {
	userID := c.GetUint("user_id")
	monitorID, ok := uintParam(c.Param("id"))
	if !ok {
		badRequest(c, "invalid_monitor_id", "invalid monitor id")
		return
	}

	monitor, err := userMonitor(h.DB, userID, monitorID)
	if err != nil {
		lookupErrorResponse(c, err, "monitor_not_found", "monitor not found", "monitor_lookup_failed")
		return
	}
	monitors, err := h.monitorActivationTargets(userID, monitor)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_descendants_failed", err.Error())
		return
	}
	ids := monitorIDs(monitors)
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", false).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_pause_failed", err.Error())
		return
	}
	stopMonitors(h.Scheduler, monitors)
	c.JSON(http.StatusOK, gin.H{"message": "monitor paused"})
}

func (h *MonitorHandler) validateMonitorRequest(userID uint, monitorID uint, req CreateMonitorRequest) (*requestValidationError, error) {
	if !isValidMonitorType(req.Type) {
		return &requestValidationError{code: "invalid_monitor_type", message: "type must be http, tcp, ping, dns, push or group"}, nil
	}
	if req.GroupID == nil {
		return nil, nil
	}
	if monitorID != 0 && *req.GroupID == monitorID {
		return &requestValidationError{code: "invalid_group", message: "monitor cannot be its own group"}, nil
	}

	parent, err := userGroupMonitor(h.DB, userID, *req.GroupID)
	if err != nil {
		if !isRecordNotFound(err) {
			return nil, err
		}
		return &requestValidationError{code: "invalid_group", message: "group_id must reference a group monitor owned by the current user"}, nil
	}
	cycle, err := wouldCreateGroupCycle(h.DB, userID, monitorID, parent.ID)
	if err != nil {
		return nil, err
	}
	if monitorID != 0 && cycle {
		return &requestValidationError{code: "group_cycle", message: "group hierarchy cannot contain cycles"}, nil
	}
	return nil, nil
}

func isValidMonitorType(monitorType string) bool {
	switch monitorType {
	case model.MonitorTypeHTTP, model.MonitorTypeTCP, model.MonitorTypePing, model.MonitorTypeDNS, model.MonitorTypePush, model.MonitorTypeGroup:
		return true
	default:
		return false
	}
}

func (h *MonitorHandler) descendantMonitors(userID uint, groupID uint) ([]model.Monitor, error) {
	var children []model.Monitor
	if err := h.DB.Where("user_id = ? AND group_id = ?", userID, groupID).Find(&children).Error; err != nil {
		return nil, err
	}
	results := make([]model.Monitor, 0, len(children))
	for _, child := range children {
		results = append(results, child)
		if child.Type == model.MonitorTypeGroup {
			descendants, err := h.descendantMonitors(userID, child.ID)
			if err != nil {
				return nil, err
			}
			results = append(results, descendants...)
		}
	}
	return results, nil
}
