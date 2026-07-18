package notifier

import "testing"

func TestParseNotificationConfigEmailRecipients(t *testing.T) {
	cfg, err := ParseNotificationConfig(`{"to":"ops@example.com","cc":"dev@example.com"}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := cfg.EmailRecipients(); got != "ops@example.com,dev@example.com" {
		t.Fatalf("recipients=%q", got)
	}

	cfg, err = ParseNotificationConfig(`{"email":"legacy@example.com"}`)
	if err != nil {
		t.Fatalf("parse legacy: %v", err)
	}
	if got := cfg.EmailRecipients(); got != "legacy@example.com" {
		t.Fatalf("legacy recipients=%q", got)
	}

	cfg, err = ParseNotificationConfig(`{"cc":"only-cc@example.com"}`)
	if err != nil {
		t.Fatalf("parse cc only: %v", err)
	}
	if got := cfg.EmailRecipients(); got != "only-cc@example.com" {
		t.Fatalf("cc-only recipients=%q", got)
	}
}

func TestParseNotificationConfigTemplateAndInvalidJSON(t *testing.T) {
	cfg, err := ParseNotificationConfig(`{"subject_template":"{{NAME}} {{STATUS}}"}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := cfg.Template("subject_template", map[string]string{"NAME": "site", "STATUS": "DOWN"})
	if got != "site DOWN" {
		t.Fatalf("template=%q", got)
	}

	if _, err := ParseNotificationConfig(`{"to":`); err == nil {
		t.Fatal("invalid json should fail")
	}
}
