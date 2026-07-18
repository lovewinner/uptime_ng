package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type ImportExportHandler struct {
	DB        *gorm.DB
	Scheduler MonitorScheduler
}

func NewImportExportHandler(db *gorm.DB, scheduler MonitorScheduler) *ImportExportHandler {
	return &ImportExportHandler{DB: db, Scheduler: scheduler}
}

type ExportFile struct {
	Version       string               `json:"version"`
	ExportedAt    string               `json:"exported_at"`
	ExportedBy    string               `json:"exported_by"`
	Monitors      []ExportMonitor      `json:"monitors"`
	Notifications []ExportNotification `json:"notifications"`
}

type ExportMonitor struct {
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	Type                  string      `json:"type"`
	Active                bool        `json:"active"`
	GroupPath             []string    `json:"group_path"`
	URL                   string      `json:"url"`
	Hostname              string      `json:"hostname"`
	Port                  uint16      `json:"port"`
	Method                string      `json:"method"`
	Interval              uint32      `json:"interval"`
	Timeout               float64     `json:"timeout"`
	MaxRetries            uint32      `json:"max_retries"`
	RetryInterval         uint32      `json:"retry_interval"`
	ResendInterval        uint32      `json:"resend_interval"`
	IgnoreTLS             bool        `json:"ignore_tls"`
	UpsideDown            bool        `json:"upside_down"`
	MaxRedirects          uint32      `json:"max_redirects"`
	AcceptedStatusCodes   []string    `json:"accepted_status_codes"`
	Headers               string      `json:"headers"`
	Body                  string      `json:"body"`
	AuthMethod            string      `json:"auth_method"`
	AuthWorkstation       string      `json:"auth_workstation"`
	AuthDomain            string      `json:"auth_domain"`
	TLSCa                 string      `json:"tls_ca"`
	OAuthTokenURL         string      `json:"oauth_token_url"`
	OAuthScopes           string      `json:"oauth_scopes"`
	OAuthAuthMethod       string      `json:"oauth_auth_method"`
	OAuthAudience         string      `json:"oauth_audience"`
	Keyword               string      `json:"keyword"`
	InvertKeyword         bool        `json:"invert_keyword"`
	ExpiryNotification    bool        `json:"expiry_notification"`
	PacketSize            uint32      `json:"packet_size"`
	HTTPBodyEncoding      string      `json:"http_body_encoding"`
	RetryOnlyOnStatusCode bool        `json:"retry_only_on_status_code"`
	CacheBust             bool        `json:"cache_bust"`
	SaveResponse          bool        `json:"save_response"`
	SaveErrorResponse     bool        `json:"save_error_response"`
	ResponseMaxLength     uint32      `json:"response_max_length"`
	DNSResolveType        string      `json:"dns_resolve_type"`
	DNSResolveServer      string      `json:"dns_resolve_server"`
	PingNumeric           bool        `json:"ping_numeric"`
	PingCount             uint32      `json:"ping_count"`
	PingPerRequestTimeout uint32      `json:"ping_per_request_timeout"`
	Tags                  []ExportTag `json:"tags"`
	NotificationNames     []string    `json:"notification_names"`
}

type ExportTag struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ExportNotification struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Config string `json:"config"`
}

type ImportPreviewResponse struct {
	NewCount            int              `json:"new_count"`
	ConflictCount       int              `json:"conflict_count"`
	Conflicts           []ImportConflict `json:"conflicts"`
	NewMonitors         []ExportMonitor  `json:"new_monitors"`
	NewTags             []ExportTag      `json:"new_tags"`
	Notifications       int              `json:"notifications"`
	MaskedNotifications int              `json:"masked_notifications"`
	Summary             string           `json:"summary"`
}

type ImportConflict struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ExistingID   uint   `json:"existing_id"`
	ExistingName string `json:"existing_name"`
}

type ImportRequest struct {
	Data     ExportFile `json:"data" binding:"required"`
	Strategy string     `json:"strategy"` // skip, overwrite, copy
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (h *ImportExportHandler) ExportMonitors(c *gin.Context) {
	userID := c.GetUint("user_id")
	username, _ := c.Get("username")

	idsParam := c.Query("ids")
	var monitors []model.Monitor

	if idsParam != "" {
		var ids []int
		_ = json.Unmarshal([]byte(idsParam), &ids)
		if len(ids) > 0 {
			if err := h.DB.Where("user_id = ? AND id IN ?", userID, ids).Find(&monitors).Error; err != nil {
				errorResponse(c, http.StatusInternalServerError, "export_query_failed", err.Error())
				return
			}
		}
	} else {
		if err := h.DB.Where("user_id = ?", userID).Find(&monitors).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "export_query_failed", err.Error())
			return
		}
	}

	exportMonitors := make([]ExportMonitor, len(monitors))

	notifNameSet := make(map[string]*model.Notification)

	for i, m := range monitors {
		var tags []model.Tag
		if err := h.DB.Raw(`
			SELECT t.* FROM tags t
			JOIN monitor_tags mt ON mt.tag_id = t.id
			WHERE mt.monitor_id = ?
		`, m.ID).Scan(&tags).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "export_tags_failed", err.Error())
			return
		}

		var notifs []model.MonitorNotification
		if err := h.DB.Where("monitor_id = ?", m.ID).Find(&notifs).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "export_notifications_failed", err.Error())
			return
		}
		notifNames := make([]string, 0, len(notifs))
		for _, mn := range notifs {
			var n model.Notification
			err := h.DB.Where("id = ?", mn.NotificationID).First(&n).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "export_notification_failed", err.Error())
				return
			}
			notifNames = append(notifNames, n.Name)
			notifNameSet[n.Name] = &n
		}

		exportMonitors[i] = exportMonitorFromModel(m, tags, notifNames, userGroupPath(h.DB, userID, m.GroupID))
	}

	exportNotifs := make([]ExportNotification, 0, len(notifNameSet))
	for _, n := range notifNameSet {
		exportNotifs = append(exportNotifs, ExportNotification{
			Name:   n.Name,
			Type:   n.Type,
			Config: maskSensitive(n.Config),
		})
	}

	file := ExportFile{
		Version:       "1.0",
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		ExportedBy:    username.(string),
		Monitors:      exportMonitors,
		Notifications: exportNotifs,
	}

	c.Header("Content-Disposition", "attachment; filename=uptime_ng_export.json")
	c.JSON(http.StatusOK, file)
}

func (h *ImportExportHandler) ImportPreview(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	existingByName := map[string]model.Monitor{}
	var existing []model.Monitor
	if err := h.DB.Where("user_id = ?", userID).Find(&existing).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "import_preview_query_failed", err.Error())
		return
	}
	for _, monitor := range existing {
		existingByName[monitor.Name] = monitor
	}

	c.JSON(http.StatusOK, buildImportPreview(existingByName, req.Data))
}

func (h *ImportExportHandler) ImportExecute(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	result := ImportResult{}
	importedMonitors := make([]model.Monitor, 0, len(req.Data.Monitors))

	if err := runTransaction(h.DB, func(tx *gorm.DB) error {
		if err := importNotifications(tx, userID, req.Data.Notifications, req.Strategy); err != nil {
			result.Errors = append(result.Errors, "failed to import notifications: "+err.Error())
			return err
		}

		for _, em := range req.Data.Monitors {
			outcome, err := importMonitor(tx, userID, em, req.Strategy)
			if err != nil {
				result.Errors = append(result.Errors, "failed to import "+em.Name+": "+err.Error())
				return err
			}
			switch outcome.action {
			case importMonitorCreated:
				result.Created++
				importedMonitors = append(importedMonitors, outcome.monitor)
			case importMonitorUpdated:
				result.Updated++
				importedMonitors = append(importedMonitors, outcome.monitor)
			case importMonitorSkipped:
				result.Skipped++
			}
		}

		return nil
	}); err != nil {
		if len(result.Errors) > 0 {
			c.JSON(http.StatusInternalServerError, result)
		} else {
			errorResponse(c, http.StatusInternalServerError, "import_commit_failed", err.Error())
		}
		return
	}
	if err := syncImportedMonitorSchedulers(h.DB, h.Scheduler, importedMonitors); err != nil {
		result.Errors = append(result.Errors, "failed to sync monitor schedulers: "+err.Error())
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	result.Imported = result.Created + result.Updated
	c.JSON(http.StatusOK, result)
}

func newMonitorFromExport(userID uint, em ExportMonitor, groupID *uint) model.Monitor {
	monitor := model.Monitor{
		UserID:  userID,
		Name:    em.Name,
		GroupID: groupID,
	}
	return applyExportMonitor(monitor, em)
}

func applyExportMonitor(existing model.Monitor, em ExportMonitor) model.Monitor {
	existing.Description = em.Description
	existing.Type = em.Type
	existing.Active = em.Active
	existing.URL = em.URL
	existing.Hostname = em.Hostname
	existing.Port = em.Port
	existing.Method = em.Method
	existing.Interval = em.Interval
	existing.Timeout = em.Timeout
	existing.MaxRetries = em.MaxRetries
	existing.RetryInterval = em.RetryInterval
	existing.ResendInterval = em.ResendInterval
	existing.IgnoreTLS = em.IgnoreTLS
	existing.UpsideDown = em.UpsideDown
	existing.MaxRedirects = em.MaxRedirects
	existing.AcceptedStatusCodes = acceptedStatusCodesJSON(em.AcceptedStatusCodes)
	existing.Headers = em.Headers
	existing.Body = em.Body
	existing.AuthMethod = em.AuthMethod
	existing.AuthWorkstation = em.AuthWorkstation
	existing.AuthDomain = em.AuthDomain
	existing.TLSCa = em.TLSCa
	existing.OAuthTokenURL = em.OAuthTokenURL
	existing.OAuthScopes = em.OAuthScopes
	existing.OAuthAuthMethod = em.OAuthAuthMethod
	existing.OAuthAudience = em.OAuthAudience
	existing.Keyword = em.Keyword
	existing.InvertKeyword = em.InvertKeyword
	existing.ExpiryNotification = em.ExpiryNotification
	existing.PacketSize = em.PacketSize
	existing.HTTPBodyEncoding = em.HTTPBodyEncoding
	existing.RetryOnlyOnStatusCode = em.RetryOnlyOnStatusCode
	existing.CacheBust = em.CacheBust
	existing.SaveResponse = em.SaveResponse
	existing.SaveErrorResponse = em.SaveErrorResponse
	existing.ResponseMaxLength = em.ResponseMaxLength
	existing.DNSResolveType = em.DNSResolveType
	existing.DNSResolveServer = em.DNSResolveServer
	existing.PingNumeric = em.PingNumeric
	existing.PingCount = em.PingCount
	existing.PingPerRequestTimeout = em.PingPerRequestTimeout
	return existing
}

func ensureGroupPath(tx *gorm.DB, userID uint, path []string) (*uint, error) {
	var parentID *uint
	for _, rawName := range path {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		var group model.Monitor
		query := tx.Where("user_id = ? AND name = ? AND type = ?", userID, name, model.MonitorTypeGroup)
		if parentID == nil {
			query = query.Where("group_id IS NULL")
		} else {
			query = query.Where("group_id = ?", *parentID)
		}
		err := query.First(&group).Error
		if err == nil {
			id := group.ID
			parentID = &id
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		group = model.Monitor{
			UserID:              userID,
			Name:                name,
			Type:                model.MonitorTypeGroup,
			GroupID:             parentID,
			Active:              true,
			Interval:            model.DefaultInterval,
			Timeout:             model.DefaultTimeout,
			AcceptedStatusCodes: `["200-299"]`,
			ExpiryNotification:  true,
			ResponseMaxLength:   model.DefaultResponseMaxLen,
			PingCount:           model.DefaultPingCount,
		}
		if err := tx.Create(&group).Error; err != nil {
			return nil, err
		}
		id := group.ID
		parentID = &id
	}
	return parentID, nil
}

func addTagIfNew(preview *ImportPreviewResponse, et ExportTag) {
	for _, existing := range preview.NewTags {
		if existing.Name == et.Name {
			return
		}
	}
	preview.NewTags = append(preview.NewTags, et)
}

func maskSensitive(config string) string {
	var value any
	if err := json.Unmarshal([]byte(config), &value); err != nil {
		return config
	}
	masked := maskJSONValue(value)
	bytes, err := json.Marshal(masked)
	if err != nil {
		return config
	}
	return string(bytes)
}

func maskJSONValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			if isSensitiveKey(key) {
				out[key] = "***"
			} else {
				out[key] = maskJSONValue(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = maskJSONValue(val)
		}
		return out
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, part := range []string{"password", "secret", "token", "webhook", "key"} {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

func containsMaskedValue(config string) bool {
	return strings.Contains(config, `"***"`) || strings.Contains(config, ":***")
}
