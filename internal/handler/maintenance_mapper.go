package handler

import (
	"time"

	"uptime_ng/internal/model"
)

func maintenanceWindowFromRequest(req MaintenanceRequest, userID uint, existing *model.MaintenanceWindow) (model.MaintenanceWindow, *requestValidationError) {
	start, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return model.MaintenanceWindow{}, &requestValidationError{code: "invalid_start_at", message: "start_at must be RFC3339"}
	}
	end, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return model.MaintenanceWindow{}, &requestValidationError{code: "invalid_end_at", message: "end_at must be RFC3339"}
	}
	if !end.After(start) {
		return model.MaintenanceWindow{}, &requestValidationError{code: "invalid_time_range", message: "end_at must be after start_at"}
	}

	active := true
	if existing != nil {
		active = existing.Active
	}
	if req.Active != nil {
		active = *req.Active
	}

	window := model.MaintenanceWindow{UserID: userID}
	if existing != nil {
		window = *existing
	}
	window.Name = req.Name
	window.Description = req.Description
	window.MonitorID = req.MonitorID
	window.StartAt = start
	window.EndAt = end
	window.Active = active
	return window, nil
}
