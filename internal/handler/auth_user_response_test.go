package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"uptime_ng/internal/model"
)

func TestUserProfileResponseOmitsPassword(t *testing.T) {
	resp := userProfileResponse(model.User{
		ID:       7,
		Username: "alice",
		Password: "secret-hash",
		Role:     model.RoleAdmin,
		Active:   true,
	})
	if resp.ID != 7 || resp.Username != "alice" || resp.Role != model.RoleAdmin {
		t.Fatalf("profile=%+v", resp)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "secret-hash") || strings.Contains(string(data), "password") {
		t.Fatalf("password leaked: %s", string(data))
	}
}

func TestUserListResponses(t *testing.T) {
	users := []model.User{
		{ID: 1, Username: "admin", Role: model.RoleAdmin, Active: true},
		{ID: 2, Username: "user", Role: model.RoleUser, Active: false},
	}
	resp := userListResponses(users)
	if len(resp) != 2 {
		t.Fatalf("len=%d", len(resp))
	}
	if resp[0].Username != "admin" || !resp[0].Active {
		t.Fatalf("first=%+v", resp[0])
	}
	if resp[1].Username != "user" || resp[1].Active {
		t.Fatalf("second=%+v", resp[1])
	}
}
