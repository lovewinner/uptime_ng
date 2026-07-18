package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"uptime_ng/internal/model"
)

func TestTokenResponseFromUser(t *testing.T) {
	resp := tokenResponseFromUser(model.User{
		ID:       7,
		Username: "admin",
		Password: "secret-hash",
		Role:     model.RoleAdmin,
	}, "jwt-token")

	if resp.Token != "jwt-token" || resp.UserID != 7 || resp.Username != "admin" || resp.Role != model.RoleAdmin {
		t.Fatalf("response=%+v", resp)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) == "" || !json.Valid(data) {
		t.Fatalf("invalid json=%s", string(data))
	}
	if strings.Contains(string(data), "secret-hash") {
		t.Fatalf("password hash leaked: %s", string(data))
	}
}
