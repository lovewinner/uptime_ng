package handler

import (
	"encoding/json"
	"strconv"

	"uptime_ng/internal/model"
)

func exportMonitorFromModel(m model.Monitor, tags []model.Tag, notificationNames []string, groupPath []string) ExportMonitor {
	return ExportMonitor{
		Name:                  m.Name,
		Description:           m.Description,
		Type:                  m.Type,
		Active:                m.Active,
		GroupPath:             groupPath,
		URL:                   m.URL,
		Hostname:              m.Hostname,
		Port:                  m.Port,
		Method:                m.Method,
		Interval:              m.Interval,
		Timeout:               m.Timeout,
		MaxRetries:            m.MaxRetries,
		RetryInterval:         m.RetryInterval,
		ResendInterval:        m.ResendInterval,
		IgnoreTLS:             m.IgnoreTLS,
		UpsideDown:            m.UpsideDown,
		MaxRedirects:          m.MaxRedirects,
		AcceptedStatusCodes:   acceptedStatusCodesFromJSON(m.AcceptedStatusCodes),
		Headers:               m.Headers,
		Body:                  m.Body,
		AuthMethod:            m.AuthMethod,
		AuthWorkstation:       m.AuthWorkstation,
		AuthDomain:            m.AuthDomain,
		TLSCa:                 m.TLSCa,
		OAuthTokenURL:         m.OAuthTokenURL,
		OAuthScopes:           m.OAuthScopes,
		OAuthAuthMethod:       m.OAuthAuthMethod,
		OAuthAudience:         m.OAuthAudience,
		Keyword:               m.Keyword,
		InvertKeyword:         m.InvertKeyword,
		ExpiryNotification:    m.ExpiryNotification,
		PacketSize:            m.PacketSize,
		HTTPBodyEncoding:      m.HTTPBodyEncoding,
		RetryOnlyOnStatusCode: m.RetryOnlyOnStatusCode,
		CacheBust:             m.CacheBust,
		SaveResponse:          m.SaveResponse,
		SaveErrorResponse:     m.SaveErrorResponse,
		ResponseMaxLength:     m.ResponseMaxLength,
		DNSResolveType:        m.DNSResolveType,
		DNSResolveServer:      m.DNSResolveServer,
		PingNumeric:           m.PingNumeric,
		PingCount:             m.PingCount,
		PingPerRequestTimeout: m.PingPerRequestTimeout,
		Tags:                  exportTagsFromModels(tags),
		NotificationNames:     notificationNames,
	}
}

func acceptedStatusCodesFromJSON(raw string) []string {
	var acceptedCodes []string
	_ = json.Unmarshal([]byte(raw), &acceptedCodes)
	if acceptedCodes == nil {
		return []string{"200-299"}
	}
	return acceptedCodes
}

func exportTagsFromModels(tags []model.Tag) []ExportTag {
	exportTags := make([]ExportTag, len(tags))
	for i, tag := range tags {
		exportTags[i] = ExportTag{Name: tag.Name, Color: tag.Color}
	}
	return exportTags
}

func buildImportPreview(existingByName map[string]model.Monitor, file ExportFile) ImportPreviewResponse {
	preview := ImportPreviewResponse{}
	for _, em := range file.Monitors {
		if existing, ok := existingByName[em.Name]; ok {
			preview.ConflictCount++
			preview.Conflicts = append(preview.Conflicts, ImportConflict{
				Name:         em.Name,
				Type:         em.Type,
				ExistingID:   existing.ID,
				ExistingName: existing.Name,
			})
			continue
		}
		preview.NewCount++
		preview.NewMonitors = append(preview.NewMonitors, em)
		for _, tag := range em.Tags {
			addTagIfNew(&preview, tag)
		}
	}
	for _, en := range file.Notifications {
		if en.Name == "" {
			continue
		}
		preview.Notifications++
		if containsMaskedValue(en.Config) {
			preview.MaskedNotifications++
		}
	}
	if preview.ConflictCount > 0 {
		preview.Summary = "found " + strconv.Itoa(preview.ConflictCount) + " conflicts, please choose a strategy"
	} else {
		preview.Summary = "all " + strconv.Itoa(preview.NewCount) + " monitors are new, ready to import"
	}
	return preview
}
