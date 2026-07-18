package notifier

import (
	"strings"
	"testing"

	"uptime_ng/internal/model"
)

func TestBuildEmailAlertContentDefaults(t *testing.T) {
	monitor := &model.Monitor{Name: "site", Type: model.MonitorTypeHTTP}
	content, ok := BuildEmailAlertContent(monitor, NotificationConfig{"to": "ops@example.com"}, false, "timeout")
	if !ok {
		t.Fatalf("expected content")
	}
	if content.To != "ops@example.com" {
		t.Fatalf("to=%q", content.To)
	}
	if content.Subject != "[uptime_ng] site - DOWN" {
		t.Fatalf("subject=%q", content.Subject)
	}
	if !strings.Contains(content.Body, "timeout") || !strings.Contains(content.Body, "site") {
		t.Fatalf("body=%q", content.Body)
	}
}

func TestBuildEmailAlertContentUsesTemplates(t *testing.T) {
	monitor := &model.Monitor{Name: "site", Type: model.MonitorTypeHTTP}
	content, ok := BuildEmailAlertContent(monitor, NotificationConfig{
		"to":               "ops@example.com",
		"subject_template": "{{NAME}} {{STATUS}}",
		"body_template":    "{{TYPE}} {{MSG}}",
	}, true, "recovered")
	if !ok {
		t.Fatalf("expected content")
	}
	if content.Subject != "site UP (已恢复)" {
		t.Fatalf("subject=%q", content.Subject)
	}
	if content.Body != "http recovered" {
		t.Fatalf("body=%q", content.Body)
	}
}

func TestBuildEmailAlertContentRequiresRecipient(t *testing.T) {
	if _, ok := BuildEmailAlertContent(&model.Monitor{}, NotificationConfig{}, false, "down"); ok {
		t.Fatalf("expected missing recipient")
	}
}
