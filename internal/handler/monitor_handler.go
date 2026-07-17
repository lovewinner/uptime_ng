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
	DNSResolveType        string   `json:"dns_resolve_type"`
	DNSResolveServer      string   `json:"dns_resolve_server"`
	PacketSize            uint32   `json:"packet_size"`
	ExpiryNotification    *bool    `json:"expiry_notification"`
	HTTPBodyEncoding      string   `json:"http_body_encoding"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Interval < model.MinInterval {
		req.Interval = model.DefaultInterval
	}
	if req.Timeout <= 0 {
		req.Timeout = model.DefaultTimeout
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
		DNSResolveType:        req.DNSResolveType,
		DNSResolveServer:      req.DNSResolveServer,
		PacketSize:            req.PacketSize,
		ExpiryNotification:    expiryNotification,
		HTTPBodyEncoding:      req.HTTPBodyEncoding,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Interval < model.MinInterval {
		req.Interval = model.DefaultInterval
	}
	if req.Timeout <= 0 {
		req.Timeout = model.DefaultTimeout
	}

	monitor.Name = req.Name
	monitor.Description = req.Description
	monitor.Type = req.Type
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
	monitor.IgnoreTLS = req.IgnoreTLS
	monitor.UpsideDown = req.UpsideDown
	monitor.Keyword = req.Keyword
	monitor.InvertKeyword = req.InvertKeyword
	monitor.MaxRedirects = req.MaxRedirects
	monitor.DNSResolveType = req.DNSResolveType
	monitor.DNSResolveServer = req.DNSResolveServer
	monitor.PacketSize = req.PacketSize
	monitor.PingNumeric = req.PingNumeric
	monitor.PingCount = req.PingCount
	monitor.PingPerRequestTimeout = req.PingPerRequestTimeout

	if len(req.AcceptedStatusCodes) > 0 {
		bytes, _ := json.Marshal(req.AcceptedStatusCodes)
		monitor.AcceptedStatusCodes = string(bytes)
	}

	tx := h.DB.Begin()
	if err := tx.Save(&monitor).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	tx := h.DB.Begin()
	tx.Where("monitor_id = ?", monitorID).Delete(&model.Heartbeat{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorNotification{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.MonitorTag{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatMinutely{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatHourly{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.StatDaily{})
	tx.Where("monitor_id = ?", monitorID).Delete(&model.Incident{})
	tx.Delete(&monitor)
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}
	if err := h.DB.Model(&monitor).Update("active", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	monitor.Active = true
	if h.Scheduler != nil {
		h.Scheduler.StartMonitor(&monitor)
	}
	c.JSON(http.StatusOK, gin.H{"message": "monitor resumed"})
}

func (h *MonitorHandler) Pause(c *gin.Context) {
	userID, _ := c.Get("user_id")
	monitorID := c.Param("id")

	var monitor model.Monitor
	if err := h.DB.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}
	if err := h.DB.Model(&monitor).Update("active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.Scheduler != nil {
		h.Scheduler.StopMonitor(monitor.ID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "monitor paused"})
}
