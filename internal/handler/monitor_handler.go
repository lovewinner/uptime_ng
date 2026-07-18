package handler

import (
	"encoding/json"
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
	if req.Interval < model.MinInterval {
		req.Interval = model.DefaultInterval
	}
	if req.Timeout <= 0 {
		req.Timeout = model.DefaultTimeout
	}
	if req.ResponseMaxLength == 0 {
		req.ResponseMaxLength = model.DefaultResponseMaxLen
	}
	if req.MaxRedirects == 0 {
		req.MaxRedirects = model.DefaultHTTPMaxRedirects
	}
	if err := h.validateMonitorRequest(userID.(uint), 0, req); err != nil {
		badRequest(c, err.code, err.message)
		return
	}

	var acceptedStatusCodesJSON string
	if len(req.AcceptedStatusCodes) > 0 {
		bytes, _ := json.Marshal(req.AcceptedStatusCodes)
		acceptedStatusCodesJSON = string(bytes)
	} else {
		acceptedStatusCodesJSON = `["200-299"]`
	}

	expiryNotification := true
	if req.ExpiryNotification != nil {
		expiryNotification = *req.ExpiryNotification
	}

	monitor := model.Monitor{
		UserID:                userID.(uint),
		Name:                  req.Name,
		Description:           req.Description,
		Type:                  req.Type,
		GroupID:               req.GroupID,
		Active:                true,
		URL:                   req.URL,
		Hostname:              req.Hostname,
		Port:                  req.Port,
		Method:                req.Method,
		Interval:              req.Interval,
		Timeout:               req.Timeout,
		MaxRetries:            req.MaxRetries,
		RetryInterval:         req.RetryInterval,
		ResendInterval:        req.ResendInterval,
		Headers:               req.Headers,
		Body:                  req.Body,
		AcceptedStatusCodes:   acceptedStatusCodesJSON,
		Keyword:               req.Keyword,
		InvertKeyword:         req.InvertKeyword,
		IgnoreTLS:             req.IgnoreTLS,
		UpsideDown:            req.UpsideDown,
		MaxRedirects:          req.MaxRedirects,
		AuthMethod:            req.AuthMethod,
		BasicAuthUser:         req.BasicAuthUser,
		BasicAuthPass:         req.BasicAuthPass,
		BearerToken:           req.BearerToken,
		AuthWorkstation:       req.AuthWorkstation,
		AuthDomain:            req.AuthDomain,
		TLSKey:                req.TLSKey,
		TLSCert:               req.TLSCert,
		TLSCa:                 req.TLSCa,
		OAuthClientID:         req.OAuthClientID,
		OAuthClientSecret:     req.OAuthClientSecret,
		OAuthTokenURL:         req.OAuthTokenURL,
		OAuthScopes:           req.OAuthScopes,
		OAuthAuthMethod:       req.OAuthAuthMethod,
		OAuthAudience:         req.OAuthAudience,
		DNSResolveType:        req.DNSResolveType,
		DNSResolveServer:      req.DNSResolveServer,
		PacketSize:            req.PacketSize,
		ExpiryNotification:    expiryNotification,
		HTTPBodyEncoding:      req.HTTPBodyEncoding,
		RetryOnlyOnStatusCode: req.RetryOnlyOnStatusCode,
		CacheBust:             req.CacheBust,
		SaveResponse:          req.SaveResponse,
		SaveErrorResponse:     req.SaveErrorResponse,
		ResponseMaxLength:     req.ResponseMaxLength,
		PingNumeric:           req.PingNumeric,
		PingCount:             req.PingCount,
		PingPerRequestTimeout: req.PingPerRequestTimeout,
	}

	if monitor.Port == 0 && req.Port == 0 {
		monitor.Port = 0
	} else {
		monitor.Port = req.Port
	}

	tx := h.DB.Begin()

	if err := tx.Create(&monitor).Error; err != nil {
		tx.Rollback()
		errorResponse(c, http.StatusInternalServerError, "monitor_create_failed", err.Error())
		return
	}

	for _, nid := range req.NotificationIDs {
		mn := model.MonitorNotification{MonitorID: monitor.ID, NotificationID: nid}
		tx.Create(&mn)
	}

	for i, tagName := range req.TagNames {
		if tagName == "" {
			continue
		}
		var tag model.Tag
		if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
			color := "#409EFF"
			if i < len(req.TagColors) && req.TagColors[i] != "" {
				color = req.TagColors[i]
			}
			tag = model.Tag{Name: tagName, Color: color}
			tx.Create(&tag)
		}
		mt := model.MonitorTag{MonitorID: monitor.ID, TagID: tag.ID, Value: tagName}
		tx.Create(&mt)
	}

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
		var tags []model.Tag
		h.DB.Raw(`
			SELECT t.* FROM tags t
			JOIN monitor_tags mt ON mt.tag_id = t.id
			WHERE mt.monitor_id = ?
		`, m.ID).Scan(&tags)

		var notifs []model.MonitorNotification
		h.DB.Where("monitor_id = ?", m.ID).Find(&notifs)
		notifIDs := make([]uint, len(notifs))
		for j, n := range notifs {
			notifIDs[j] = n.NotificationID
		}

		results[i] = gin.H{
			"monitor":          m,
			"tags":             tags,
			"notification_ids": notifIDs,
		}
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

	var tags []model.Tag
	h.DB.Raw(`
		SELECT t.* FROM tags t
		JOIN monitor_tags mt ON mt.tag_id = t.id
		WHERE mt.monitor_id = ?
	`, monitor.ID).Scan(&tags)

	var notifs []model.MonitorNotification
	h.DB.Where("monitor_id = ?", monitor.ID).Find(&notifs)
	notifIDs := make([]uint, len(notifs))
	for j, n := range notifs {
		notifIDs[j] = n.NotificationID
	}

	c.JSON(http.StatusOK, gin.H{
		"monitor":          monitor,
		"tags":             tags,
		"notification_ids": notifIDs,
	})
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
	if req.Interval < model.MinInterval {
		req.Interval = model.DefaultInterval
	}
	if req.Timeout <= 0 {
		req.Timeout = model.DefaultTimeout
	}
	if req.ResponseMaxLength == 0 {
		req.ResponseMaxLength = model.DefaultResponseMaxLen
	}
	if req.MaxRedirects == 0 {
		req.MaxRedirects = model.DefaultHTTPMaxRedirects
	}
	if err := h.validateMonitorRequest(userID.(uint), monitor.ID, req); err != nil {
		badRequest(c, err.code, err.message)
		return
	}

	wasGroup := monitor.Type == model.MonitorTypeGroup
	monitor.Name = req.Name
	monitor.Description = req.Description
	monitor.Type = req.Type
	monitor.GroupID = req.GroupID
	monitor.URL = req.URL
	monitor.Hostname = req.Hostname
	monitor.Port = req.Port
	monitor.Method = req.Method
	monitor.Interval = req.Interval
	monitor.Timeout = req.Timeout
	monitor.MaxRetries = req.MaxRetries
	monitor.RetryInterval = req.RetryInterval
	monitor.ResendInterval = req.ResendInterval
	monitor.Headers = req.Headers
	monitor.Body = req.Body
	monitor.AuthMethod = req.AuthMethod
	monitor.BasicAuthUser = req.BasicAuthUser
	monitor.BasicAuthPass = req.BasicAuthPass
	monitor.BearerToken = req.BearerToken
	monitor.AuthWorkstation = req.AuthWorkstation
	monitor.AuthDomain = req.AuthDomain
	monitor.TLSKey = req.TLSKey
	monitor.TLSCert = req.TLSCert
	monitor.TLSCa = req.TLSCa
	monitor.OAuthClientID = req.OAuthClientID
	monitor.OAuthClientSecret = req.OAuthClientSecret
	monitor.OAuthTokenURL = req.OAuthTokenURL
	monitor.OAuthScopes = req.OAuthScopes
	monitor.OAuthAuthMethod = req.OAuthAuthMethod
	monitor.OAuthAudience = req.OAuthAudience
	monitor.IgnoreTLS = req.IgnoreTLS
	monitor.UpsideDown = req.UpsideDown
	monitor.Keyword = req.Keyword
	monitor.InvertKeyword = req.InvertKeyword
	monitor.MaxRedirects = req.MaxRedirects
	monitor.DNSResolveType = req.DNSResolveType
	monitor.DNSResolveServer = req.DNSResolveServer
	monitor.PacketSize = req.PacketSize
	if req.ExpiryNotification != nil {
		monitor.ExpiryNotification = *req.ExpiryNotification
	}
	monitor.HTTPBodyEncoding = req.HTTPBodyEncoding
	monitor.RetryOnlyOnStatusCode = req.RetryOnlyOnStatusCode
	monitor.CacheBust = req.CacheBust
	monitor.SaveResponse = req.SaveResponse
	monitor.SaveErrorResponse = req.SaveErrorResponse
	monitor.ResponseMaxLength = req.ResponseMaxLength
	monitor.PingNumeric = req.PingNumeric
	monitor.PingCount = req.PingCount
	monitor.PingPerRequestTimeout = req.PingPerRequestTimeout

	if len(req.AcceptedStatusCodes) > 0 {
		bytes, _ := json.Marshal(req.AcceptedStatusCodes)
		monitor.AcceptedStatusCodes = string(bytes)
	}

	tx := h.DB.Begin()
	if wasGroup && monitor.Type != model.MonitorTypeGroup {
		tx.Model(&model.Monitor{}).Where("group_id = ?", monitor.ID).Update("group_id", nil)
	}
	if err := tx.Save(&monitor).Error; err != nil {
		tx.Rollback()
		errorResponse(c, http.StatusInternalServerError, "monitor_update_failed", err.Error())
		return
	}

	if req.NotificationIDs != nil {
		tx.Where("monitor_id = ?", monitor.ID).Delete(&model.MonitorNotification{})
		for _, nid := range req.NotificationIDs {
			mn := model.MonitorNotification{MonitorID: monitor.ID, NotificationID: nid}
			tx.Create(&mn)
		}
	}

	if req.TagNames != nil {
		tx.Where("monitor_id = ?", monitor.ID).Delete(&model.MonitorTag{})
		for i, tagName := range req.TagNames {
			if tagName == "" {
				continue
			}
			var tag model.Tag
			if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
				color := "#409EFF"
				if i < len(req.TagColors) && req.TagColors[i] != "" {
					color = req.TagColors[i]
				}
				tag = model.Tag{Name: tagName, Color: color}
				tx.Create(&tag)
			}
			tx.Create(&model.MonitorTag{MonitorID: monitor.ID, TagID: tag.ID, Value: tagName})
		}
	}

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
	monitors := []model.Monitor{monitor}
	if monitor.Type == model.MonitorTypeGroup {
		monitors = append(monitors, h.descendantMonitors(userID.(uint), monitor.ID)...)
	}
	ids := make([]uint, 0, len(monitors))
	for _, m := range monitors {
		ids = append(ids, m.ID)
	}
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", true).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_resume_failed", err.Error())
		return
	}
	if h.Scheduler != nil {
		for i := range monitors {
			monitors[i].Active = true
			h.Scheduler.RestartMonitor(&monitors[i])
		}
	}
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
	monitors := []model.Monitor{monitor}
	if monitor.Type == model.MonitorTypeGroup {
		monitors = append(monitors, h.descendantMonitors(userID.(uint), monitor.ID)...)
	}
	ids := make([]uint, 0, len(monitors))
	for _, m := range monitors {
		ids = append(ids, m.ID)
	}
	if err := h.DB.Model(&model.Monitor{}).Where("id IN ?", ids).Update("active", false).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_pause_failed", err.Error())
		return
	}
	if h.Scheduler != nil {
		for _, m := range monitors {
			h.Scheduler.StopMonitor(m.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "monitor paused"})
}

type monitorValidationError struct {
	code    string
	message string
}

func (h *MonitorHandler) validateMonitorRequest(userID uint, monitorID uint, req CreateMonitorRequest) *monitorValidationError {
	if !isValidMonitorType(req.Type) {
		return &monitorValidationError{code: "invalid_monitor_type", message: "type must be http, tcp, ping, dns, push or group"}
	}
	if req.GroupID == nil {
		return nil
	}
	if monitorID != 0 && *req.GroupID == monitorID {
		return &monitorValidationError{code: "invalid_group", message: "monitor cannot be its own group"}
	}

	var parent model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ? AND type = ?", *req.GroupID, userID, model.MonitorTypeGroup).First(&parent).Error; err != nil {
		return &monitorValidationError{code: "invalid_group", message: "group_id must reference a group monitor owned by the current user"}
	}
	if monitorID != 0 && h.wouldCreateGroupCycle(userID, monitorID, parent.ID) {
		return &monitorValidationError{code: "group_cycle", message: "group hierarchy cannot contain cycles"}
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
