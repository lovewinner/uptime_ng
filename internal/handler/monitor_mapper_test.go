package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestNewMonitorFromRequestAppliesDefaultsAndFields(t *testing.T) {
	expiry := false
	req := CreateMonitorRequest{
		Name:                  "site",
		Type:                  model.MonitorTypeHTTP,
		URL:                   "https://example.com",
		Interval:              1,
		Timeout:               0,
		MaxRedirects:          0,
		ResponseMaxLength:     0,
		AcceptedStatusCodes:   []string{"200", "204"},
		ExpiryNotification:    &expiry,
		AuthMethod:            "ntlm",
		AuthDomain:            "DOMAIN",
		AuthWorkstation:       "WORKSTATION",
		RetryOnlyOnStatusCode: true,
	}

	normalizeMonitorRequest(&req)
	monitor := newMonitorFromRequest(7, req)

	if monitor.UserID != 7 || !monitor.Active {
		t.Fatalf("identity fields not initialized: user_id=%d active=%v", monitor.UserID, monitor.Active)
	}
	if monitor.Interval != model.DefaultInterval {
		t.Fatalf("interval=%d want %d", monitor.Interval, model.DefaultInterval)
	}
	if monitor.Timeout != model.DefaultTimeout {
		t.Fatalf("timeout=%f want %f", monitor.Timeout, model.DefaultTimeout)
	}
	if monitor.MaxRedirects != model.DefaultHTTPMaxRedirects {
		t.Fatalf("max_redirects=%d want %d", monitor.MaxRedirects, model.DefaultHTTPMaxRedirects)
	}
	if monitor.ResponseMaxLength != model.DefaultResponseMaxLen {
		t.Fatalf("response_max_length=%d want %d", monitor.ResponseMaxLength, model.DefaultResponseMaxLen)
	}
	if monitor.AcceptedStatusCodes != `["200","204"]` {
		t.Fatalf("accepted_status_codes=%s", monitor.AcceptedStatusCodes)
	}
	if monitor.ExpiryNotification {
		t.Fatal("expiry_notification should follow explicit false")
	}
	if monitor.AuthMethod != "ntlm" || monitor.AuthDomain != "DOMAIN" || monitor.AuthWorkstation != "WORKSTATION" {
		t.Fatalf("auth fields not mapped: %#v", monitor)
	}
	if !monitor.RetryOnlyOnStatusCode {
		t.Fatal("retry_only_on_status_code not mapped")
	}
}

func TestApplyMonitorRequestPreservesOptionalUpdateFields(t *testing.T) {
	monitor := model.Monitor{
		AcceptedStatusCodes: `["201"]`,
		ExpiryNotification:  false,
	}
	req := CreateMonitorRequest{
		Name:              "renamed",
		Type:              model.MonitorTypeTCP,
		Interval:          model.DefaultInterval,
		Timeout:           model.DefaultTimeout,
		MaxRedirects:      model.DefaultHTTPMaxRedirects,
		ResponseMaxLength: model.DefaultResponseMaxLen,
	}

	applyMonitorRequest(&monitor, req, false)

	if monitor.Name != "renamed" || monitor.Type != model.MonitorTypeTCP {
		t.Fatalf("basic fields not mapped: %#v", monitor)
	}
	if monitor.AcceptedStatusCodes != `["201"]` {
		t.Fatalf("accepted_status_codes overwritten: %s", monitor.AcceptedStatusCodes)
	}
	if monitor.ExpiryNotification {
		t.Fatal("expiry_notification should be preserved when omitted")
	}

	enabled := true
	req.ExpiryNotification = &enabled
	req.AcceptedStatusCodes = []string{"200-299"}
	applyMonitorRequest(&monitor, req, false)

	if !monitor.ExpiryNotification {
		t.Fatal("expiry_notification should be updated when provided")
	}
	if monitor.AcceptedStatusCodes != `["200-299"]` {
		t.Fatalf("accepted_status_codes not updated: %s", monitor.AcceptedStatusCodes)
	}
}

func TestNewMonitorFromExportUsesSharedStatusCodeDefault(t *testing.T) {
	monitor := newMonitorFromExport(11, ExportMonitor{
		Name:   "imported",
		Type:   model.MonitorTypeHTTP,
		Active: true,
	}, nil)

	if monitor.UserID != 11 || monitor.Name != "imported" {
		t.Fatalf("identity fields not initialized: %#v", monitor)
	}
	if monitor.AcceptedStatusCodes != `["200-299"]` {
		t.Fatalf("accepted_status_codes=%s", monitor.AcceptedStatusCodes)
	}
}
