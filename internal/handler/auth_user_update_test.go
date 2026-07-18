package handler

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestPlanUserUpdateRejectsInvalidChanges(t *testing.T) {
	admin := model.User{ID: 1, Role: model.RoleAdmin, Active: true}

	tests := []struct {
		name            string
		req             UpdateUserRequest
		currentUserID   uint
		lastActiveAdmin bool
		wantCode        string
	}{
		{
			name:     "invalid role",
			req:      UpdateUserRequest{Role: stringPtr("owner")},
			wantCode: "invalid_role",
		},
		{
			name:            "demote last active admin",
			req:             UpdateUserRequest{Role: stringPtr(model.RoleUser)},
			lastActiveAdmin: true,
			wantCode:        "last_admin",
		},
		{
			name:          "deactivate self",
			req:           UpdateUserRequest{Active: boolPtr(false)},
			currentUserID: 1,
			wantCode:      "self_deactivate",
		},
		{
			name:            "deactivate last active admin",
			req:             UpdateUserRequest{Active: boolPtr(false)},
			lastActiveAdmin: true,
			wantCode:        "last_admin",
		},
		{
			name:     "short password",
			req:      UpdateUserRequest{Password: stringPtr("short")},
			wantCode: "invalid_password",
		},
		{
			name:     "empty update",
			req:      UpdateUserRequest{},
			wantCode: "empty_update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, validationErr := planUserUpdate(tt.req, admin, tt.currentUserID, tt.lastActiveAdmin)
			if validationErr == nil {
				t.Fatalf("expected validation error")
			}
			if validationErr.code != tt.wantCode {
				t.Fatalf("code=%s want %s", validationErr.code, tt.wantCode)
			}
		})
	}
}

func TestPlanUserUpdateBuildsFields(t *testing.T) {
	target := model.User{ID: 2, Role: model.RoleUser, Active: true}
	req := UpdateUserRequest{
		Role:     stringPtr(model.RoleAdmin),
		Active:   boolPtr(false),
		Password: stringPtr("new-password"),
	}

	plan, validationErr := planUserUpdate(req, target, 1, false)
	if validationErr != nil {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}

	updates := plan.fields("hashed-password")
	if updates["role"] != model.RoleAdmin {
		t.Fatalf("role=%v want %s", updates["role"], model.RoleAdmin)
	}
	if updates["active"] != false {
		t.Fatalf("active=%v want false", updates["active"])
	}
	if updates["password"] != "hashed-password" {
		t.Fatalf("password=%v want hashed-password", updates["password"])
	}
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
