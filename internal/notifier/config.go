package notifier

import (
	"encoding/json"
	"strings"
)

type NotificationConfig map[string]string

func ParseNotificationConfig(raw string) (NotificationConfig, error) {
	cfg := NotificationConfig{}
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c NotificationConfig) WebhookURL() string {
	return strings.TrimSpace(c["webhook_url"])
}

func (c NotificationConfig) EmailRecipients() string {
	to := strings.TrimSpace(c["email"])
	if to == "" {
		to = strings.TrimSpace(c["to"])
	}
	cc := strings.TrimSpace(c["cc"])
	if cc == "" {
		return to
	}
	if to == "" {
		return cc
	}
	return to + "," + cc
}

func (c NotificationConfig) Template(key string, vars map[string]string) string {
	result := strings.TrimSpace(c[key])
	if result == "" {
		return ""
	}
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
