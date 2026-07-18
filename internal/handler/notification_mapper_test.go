package handler

import (
	"strings"
	"testing"

	"uptime_ng/internal/model"
	"uptime_ng/internal/notifier"
)

func TestNotificationFromRequestValidatesType(t *testing.T) {
	_, validationErr := notificationFromRequest(CreateNotificationRequest{
		Name:   "bad",
		Type:   "sms",
		Config: "{}",
	}, 7)
	if validationErr == nil {
		t.Fatalf("expected validation error")
	}
	if validationErr.code != "invalid_notification_type" {
		t.Fatalf("code=%s want invalid_notification_type", validationErr.code)
	}
}

func TestApplyNotificationRequestPreservesIdentityAndActive(t *testing.T) {
	notif := model.Notification{ID: 3, UserID: 7, Name: "old", Type: model.NotificationTypeFeishu, Config: "{}", Active: false}
	validationErr := applyNotificationRequest(&notif, CreateNotificationRequest{
		Name:   "new",
		Type:   model.NotificationTypeEmail,
		Config: `{"to":"ops@example.com"}`,
	})
	if validationErr != nil {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}
	if notif.ID != 3 || notif.UserID != 7 || notif.Active {
		t.Fatalf("identity/active changed: %+v", notif)
	}
	if notif.Name != "new" || notif.Type != model.NotificationTypeEmail || notif.Config != `{"to":"ops@example.com"}` {
		t.Fatalf("notification not updated: %+v", notif)
	}
}

func TestNotificationTestTargetFromConfig(t *testing.T) {
	feishu, validationErr := notificationTestTargetFromConfig(
		model.Notification{Name: "ops", Type: model.NotificationTypeFeishu},
		notifier.NotificationConfig{"webhook_url": "https://hook"},
	)
	if validationErr != nil {
		t.Fatalf("unexpected feishu validation error: %+v", validationErr)
	}
	if feishu.webhookURL != "https://hook" || !strings.Contains(feishu.message, "ops (feishu)") {
		t.Fatalf("feishu target=%+v", feishu)
	}

	email, validationErr := notificationTestTargetFromConfig(
		model.Notification{Name: "mail", Type: model.NotificationTypeEmail},
		notifier.NotificationConfig{"to": "ops@example.com", "cc": "dev@example.com"},
	)
	if validationErr != nil {
		t.Fatalf("unexpected email validation error: %+v", validationErr)
	}
	if email.emailTo != "ops@example.com,dev@example.com" || email.emailSubject == "" || !strings.Contains(email.emailBody, "mail (email)") {
		t.Fatalf("email target=%+v", email)
	}
}

func TestNotificationTestTargetFromConfigRejectsMissingDestination(t *testing.T) {
	tests := []struct {
		name     string
		notif    model.Notification
		cfg      notifier.NotificationConfig
		wantCode string
	}{
		{
			name:     "feishu webhook",
			notif:    model.Notification{Name: "ops", Type: model.NotificationTypeFeishu},
			cfg:      notifier.NotificationConfig{},
			wantCode: "missing_webhook_url",
		},
		{
			name:     "email recipient",
			notif:    model.Notification{Name: "mail", Type: model.NotificationTypeEmail},
			cfg:      notifier.NotificationConfig{},
			wantCode: "missing_email_recipient",
		},
		{
			name:     "unsupported type",
			notif:    model.Notification{Name: "sms", Type: "sms"},
			cfg:      notifier.NotificationConfig{},
			wantCode: "unsupported_notification_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, validationErr := notificationTestTargetFromConfig(tt.notif, tt.cfg)
			if validationErr == nil {
				t.Fatalf("expected validation error")
			}
			if validationErr.code != tt.wantCode {
				t.Fatalf("code=%s want %s", validationErr.code, tt.wantCode)
			}
		})
	}
}
