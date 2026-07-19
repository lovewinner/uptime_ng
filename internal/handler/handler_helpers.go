package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

// --- Request parsing ---

func positiveIntParam(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func uintParam(value string) (uint, bool) {
	n, err := strconv.ParseUint(value, 10, 0)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

// --- Error responses ---

type requestValidationError struct {
	code    string
	message string
}

func errorResponse(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func badRequest(c *gin.Context, code string, message string) {
	errorResponse(c, http.StatusBadRequest, code, message)
}

func lookupErrorResponse(c *gin.Context, err error, notFoundCode string, notFoundMessage string, failureCode string) {
	if isRecordNotFound(err) {
		errorResponse(c, http.StatusNotFound, notFoundCode, notFoundMessage)
		return
	}
	errorResponse(c, http.StatusInternalServerError, failureCode, err.Error())
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// --- Transaction ---

func runTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// --- Resource lookup ---

func userMonitor(db *gorm.DB, userID uint, monitorID uint) (model.Monitor, error) {
	var monitor model.Monitor
	err := db.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error
	return monitor, err
}

func userGroupMonitor(db *gorm.DB, userID uint, monitorID uint) (model.Monitor, error) {
	var monitor model.Monitor
	err := db.Where("id = ? AND user_id = ? AND type = ?", monitorID, userID, model.MonitorTypeGroup).First(&monitor).Error
	return monitor, err
}

func userNotification(db *gorm.DB, userID uint, notificationID uint) (model.Notification, error) {
	var notification model.Notification
	err := db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error
	return notification, err
}

func userMaintenanceWindow(db *gorm.DB, userID uint, windowID uint) (model.MaintenanceWindow, error) {
	var window model.MaintenanceWindow
	err := db.Where("id = ? AND user_id = ?", windowID, userID).First(&window).Error
	return window, err
}

// --- Auth response helpers ---

type UserProfileResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type UserListResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

func tokenResponseFromUser(user model.User, token string) TokenResponse {
	return TokenResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}

func userProfileResponse(user model.User) UserProfileResponse {
	return UserProfileResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}

func userListResponses(users []model.User) []UserListResponse {
	results := make([]UserListResponse, len(users))
	for i, user := range users {
		results[i] = UserListResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			Active:   user.Active,
		}
	}
	return results
}

// --- Group hierarchy helpers ---

func userMonitorParentID(db *gorm.DB, userID uint, monitorID uint) (*uint, error) {
	var monitor model.Monitor
	err := db.Select("id", "group_id").Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error
	return monitor.GroupID, err
}

func wouldCreateGroupCycle(db *gorm.DB, userID uint, monitorID uint, parentID uint) (bool, error) {
	seen := map[uint]bool{}
	current := parentID
	for current != 0 {
		if current == monitorID {
			return true, nil
		}
		if seen[current] {
			return true, nil
		}
		seen[current] = true

		pid, err := userMonitorParentID(db, userID, current)
		if err != nil {
			return false, err
		}
		if pid == nil {
			return false, nil
		}
		current = *pid
	}
	return false, nil
}

func userGroupPath(db *gorm.DB, userID uint, groupID *uint) []string {
	if groupID == nil {
		return nil
	}

	path := []string{}
	seen := map[uint]bool{}
	current := *groupID
	for current != 0 {
		if seen[current] {
			break
		}
		seen[current] = true

		group, err := userGroupMonitor(db, userID, current)
		if err != nil {
			break
		}
		path = append([]string{group.Name}, path...)
		if group.GroupID == nil {
			break
		}
		current = *group.GroupID
	}
	return path
}
