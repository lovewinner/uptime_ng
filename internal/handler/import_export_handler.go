package handler

import (
	"encoding/json"
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
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	idsParam := c.Query("ids")
	var monitors []model.Monitor

	if idsParam != "" {
		var ids []int
		_ = json.Unmarshal([]byte(idsParam), &ids)
		if len(ids) > 0 {
			h.DB.Where("user_id = ? AND id IN ?", userID, ids).Find(&monitors)
		}
	} else {
		h.DB.Where("user_id = ?", userID).Find(&monitors)
	}

	exportMonitors := make([]ExportMonitor, len(monitors))

	notifNameSet := make(map[string]*model.Notification)

	for i, m := range monitors {
		var tags []model.Tag
		h.DB.Raw(`
			SELECT t.* FROM tags t
			JOIN monitor_tags mt ON mt.tag_id = t.id
			WHERE mt.monitor_id = ?
		`, m.ID).Scan(&tags)

		var notifs []model.MonitorNotification
		h.DB.Where("monitor_id = ?", m.ID).Find(&notifs)
		notifNames := make([]string, len(notifs))
		for j, mn := range notifs {
			var n model.Notification
			if err := h.DB.Where("id = ?", mn.NotificationID).First(&n).Error; err == nil {
				notifNames[j] = n.Name
				notifNameSet[n.Name] = &n
			}
		}

		exportMonitors[i] = exportMonitorFromModel(m, tags, notifNames, h.groupPath(userID.(uint), m.GroupID))
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
	userID, _ := c.Get("user_id")

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	existingByName := map[string]model.Monitor{}
	var existing []model.Monitor
	h.DB.Where("user_id = ?", userID).Find(&existing)
	for _, monitor := range existing {
		existingByName[monitor.Name] = monitor
	}

	c.JSON(http.StatusOK, buildImportPreview(existingByName, req.Data))
}

func (h *ImportExportHandler) ImportExecute(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	result := ImportResult{}
	tx := h.DB.Begin()
	importedMonitors := make([]model.Monitor, 0, len(req.Data.Monitors))

	importNotifications(tx, userID.(uint), req.Data.Notifications, req.Strategy)

	for _, em := range req.Data.Monitors {
		groupID, err := ensureGroupPath(tx, userID.(uint), em.GroupPath)
		if err != nil {
			result.Errors = append(result.Errors, "failed to prepare group for "+em.Name+": "+err.Error())
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, result)
			return
		}
		var existing model.Monitor
		err = tx.Where("user_id = ? AND name = ?", userID, em.Name).First(&existing).Error

		if err == nil {
			switch req.Strategy {
			case "skip":
				result.Skipped++
				continue
			case "overwrite":
				existing = applyExportMonitor(existing, em)
				existing.GroupID = groupID
				if err := tx.Save(&existing).Error; err != nil {
					result.Errors = append(result.Errors, "failed to update "+em.Name+": "+err.Error())
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, result)
					return
				}
				refreshTagsAndNotifs(tx, &existing, em)
				importedMonitors = append(importedMonitors, existing)
				result.Updated++
			case "copy":
				em.Name = em.Name + " (copy)"
				fallthrough
			default:
				result.Created++
				monitor := newMonitorFromExport(userID.(uint), em, groupID)
				if err := tx.Create(&monitor).Error; err != nil {
					result.Errors = append(result.Errors, "failed to create "+em.Name+": "+err.Error())
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, result)
					return
				}
				attachTagsAndNotifs(tx, &monitor, em)
				importedMonitors = append(importedMonitors, monitor)
			}
		} else {
			result.Created++
			monitor := newMonitorFromExport(userID.(uint), em, groupID)
			if err := tx.Create(&monitor).Error; err != nil {
				result.Errors = append(result.Errors, "failed to create "+em.Name+": "+err.Error())
				continue
			}
			attachTagsAndNotifs(tx, &monitor, em)
			importedMonitors = append(importedMonitors, monitor)
		}
	}

	if err := tx.Commit().Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "import_commit_failed", err.Error())
		return
	}
	syncImportedMonitorSchedulers(h.Scheduler, importedMonitors)
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

func (h *ImportExportHandler) groupPath(userID uint, groupID *uint) []string {
	if groupID == nil {
		return nil
	}
	path := []string{}
	seen := map[uint]bool{}
	current := *groupID
	for current != 0 {
		if seen[current] {
			break
		}
		seen[current] = true
		var group model.Monitor
		if err := h.DB.Select("id", "name", "group_id").Where("id = ? AND user_id = ? AND type = ?", current, userID, model.MonitorTypeGroup).First(&group).Error; err != nil {
			break
		}
		path = append([]string{group.Name}, path...)
		if group.GroupID == nil {
			break
		}
		current = *group.GroupID
	}
	return path
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
		if err := query.First(&group).Error; err == nil {
			id := group.ID
			parentID = &id
			continue
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

func attachTagsAndNotifs(tx *gorm.DB, monitor *model.Monitor, em ExportMonitor) {
	for _, et := range em.Tags {
		if et.Name == "" {
			continue
		}
		tag := findOrCreateTag(tx, et.Name, tagColor(et.Color))
		tx.Create(&model.MonitorTag{MonitorID: monitor.ID, TagID: tag.ID, Value: et.Name})
	}

	for _, nn := range em.NotificationNames {
		var notif model.Notification
		if err := tx.Where("name = ?", nn).First(&notif).Error; err == nil {
			mn := model.MonitorNotification{MonitorID: monitor.ID, NotificationID: notif.ID}
			tx.Create(&mn)
		}
	}
}

func refreshTagsAndNotifs(tx *gorm.DB, monitor *model.Monitor, em ExportMonitor) {
	tx.Where("monitor_id = ?", monitor.ID).Delete(&model.MonitorTag{})
	tx.Where("monitor_id = ?", monitor.ID).Delete(&model.MonitorNotification{})
	attachTagsAndNotifs(tx, monitor, em)
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
