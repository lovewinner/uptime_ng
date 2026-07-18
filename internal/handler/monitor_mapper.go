package handler

import (
	"encoding/json"

	"uptime_ng/internal/model"
)

func normalizeMonitorRequest(req *CreateMonitorRequest) {
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
}

func newMonitorFromRequest(userID uint, req CreateMonitorRequest) model.Monitor {
	monitor := model.Monitor{
		UserID:              userID,
		Active:              true,
		ExpiryNotification:  true,
		AcceptedStatusCodes: acceptedStatusCodesJSON(req.AcceptedStatusCodes),
	}
	applyMonitorRequest(&monitor, req, true)
	return monitor
}

func applyMonitorRequest(monitor *model.Monitor, req CreateMonitorRequest, overwriteAcceptedStatusCodes bool) {
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
	monitor.Keyword = req.Keyword
	monitor.InvertKeyword = req.InvertKeyword
	monitor.IgnoreTLS = req.IgnoreTLS
	monitor.UpsideDown = req.UpsideDown
	monitor.MaxRedirects = req.MaxRedirects

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

	if overwriteAcceptedStatusCodes || len(req.AcceptedStatusCodes) > 0 {
		monitor.AcceptedStatusCodes = acceptedStatusCodesJSON(req.AcceptedStatusCodes)
	}
}

func acceptedStatusCodesJSON(codes []string) string {
	if len(codes) == 0 {
		return `["200-299"]`
	}
	bytes, _ := json.Marshal(codes)
	return string(bytes)
}
