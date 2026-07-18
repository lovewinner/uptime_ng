package handler

import "uptime_ng/internal/model"

type userUpdatePlan struct {
	role     *string
	active   *bool
	password *string
}

func planUserUpdate(req UpdateUserRequest, target model.User, currentUserID uint, lastActiveAdmin bool) (userUpdatePlan, *requestValidationError) {
	plan := userUpdatePlan{}

	if req.Role != nil {
		if *req.Role != model.RoleAdmin && *req.Role != model.RoleUser {
			return plan, &requestValidationError{code: "invalid_role", message: "role must be admin or user"}
		}
		if target.Role == model.RoleAdmin && *req.Role != model.RoleAdmin && lastActiveAdmin {
			return plan, &requestValidationError{code: "last_admin", message: "cannot remove the last active admin"}
		}
		plan.role = req.Role
	}

	if req.Active != nil {
		if target.ID == currentUserID && !*req.Active {
			return plan, &requestValidationError{code: "self_deactivate", message: "cannot deactivate yourself"}
		}
		if target.Role == model.RoleAdmin && !*req.Active && lastActiveAdmin {
			return plan, &requestValidationError{code: "last_admin", message: "cannot deactivate the last active admin"}
		}
		plan.active = req.Active
	}

	if req.Password != nil {
		if len(*req.Password) < 6 {
			return plan, &requestValidationError{code: "invalid_password", message: "password must be at least 6 characters"}
		}
		plan.password = req.Password
	}

	if plan.empty() {
		return plan, &requestValidationError{code: "empty_update", message: "no updates provided"}
	}
	return plan, nil
}

func (p userUpdatePlan) empty() bool {
	return p.role == nil && p.active == nil && p.password == nil
}

func (p userUpdatePlan) fields(hashedPassword string) map[string]any {
	updates := map[string]any{}
	if p.role != nil {
		updates["role"] = *p.role
	}
	if p.active != nil {
		updates["active"] = *p.active
	}
	if p.password != nil {
		updates["password"] = hashedPassword
	}
	return updates
}
