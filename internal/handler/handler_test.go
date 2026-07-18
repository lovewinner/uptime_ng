package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/config"
	"uptime_ng/internal/engine"
	"uptime_ng/internal/middleware"
	"uptime_ng/internal/model"
)

type schedulerCall struct {
	action string
	id     uint
}

type fakeScheduler struct {
	calls         []schedulerCall
	startErr      error
	restartErr    error
	restartErrFor map[uint]error
}

var testDBSeq uint64

func (s *fakeScheduler) StartMonitor(m *model.Monitor) error {
	s.calls = append(s.calls, schedulerCall{action: "start", id: m.ID})
	return s.startErr
}

func (s *fakeScheduler) StopMonitor(id uint) {
	s.calls = append(s.calls, schedulerCall{action: "stop", id: id})
}

func (s *fakeScheduler) RestartMonitor(m *model.Monitor) error {
	s.calls = append(s.calls, schedulerCall{action: "restart", id: m.ID})
	if s.restartErrFor != nil {
		if err, ok := s.restartErrFor[m.ID]; ok {
			return err
		}
	}
	return s.restartErr
}

func assertSchedulerCalls(t *testing.T, scheduler *fakeScheduler, want []schedulerCall) {
	t.Helper()
	if len(scheduler.calls) != len(want) {
		t.Fatalf("scheduler calls=%+v want %+v", scheduler.calls, want)
	}
	for i := range want {
		if scheduler.calls[i] != want[i] {
			t.Fatalf("scheduler calls=%+v want %+v", scheduler.calls, want)
		}
	}
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	seq := atomic.AddUint64(&testDBSeq, 1)
	name := fmt.Sprintf("%s_%d", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()), seq)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Monitor{},
		&model.Heartbeat{},
		&model.StatMinutely{},
		&model.StatHourly{},
		&model.StatDaily{},
		&model.Notification{},
		&model.MonitorNotification{},
		&model.Tag{},
		&model.MonitorTag{},
		&model.Incident{},
		&model.MaintenanceWindow{},
		&model.SLAReport{},
		&model.Setting{},
	); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func authToken(t *testing.T, user *model.User) string {
	t.Helper()
	token, err := model.GenerateJWT(user, config.AppConfig.JWT.Secret, 72)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return token
}

func setupRouter(t *testing.T, db *gorm.DB, scheduler MonitorScheduler) (*gin.Engine, *model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHours: 72}}
	user := &model.User{Username: "admin", Password: "hash", Role: model.RoleAdmin, Active: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	r := gin.New()
	auth := NewAuthHandler(db)
	r.POST("/api/auth/register", auth.Register)
	r.POST("/api/auth/login", auth.Login)
	hub := NewWSHub()
	go hub.Run()
	r.GET("/api/ws", middleware.WSAuthRequired(), hub.HandleWebSocket)
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	monitor := NewMonitorHandler(db, scheduler)
	api.GET("/monitors", monitor.List)
	api.POST("/monitors", monitor.Create)
	api.GET("/monitors/:id", monitor.Get)
	api.PUT("/monitors/:id", monitor.Update)
	api.DELETE("/monitors/:id", monitor.Delete)
	api.POST("/monitors/:id/resume", monitor.Resume)
	api.POST("/monitors/:id/pause", monitor.Pause)
	api.PATCH("/auth/users/:id", middleware.AdminRequired(), auth.UpdateUser)
	ie := NewImportExportHandler(db, scheduler)
	api.GET("/monitors/export", ie.ExportMonitors)
	api.POST("/monitors/import", ie.ImportExecute)
	notif := NewNotificationHandler(db)
	api.GET("/notifications/:id", notif.Get)
	api.PUT("/notifications/:id", notif.Update)
	api.DELETE("/notifications/:id", notif.Delete)
	api.POST("/notifications/:id/test", notif.Test)
	maintenance := NewMaintenanceHandler(db)
	api.GET("/maintenance", maintenance.List)
	api.POST("/maintenance", maintenance.Create)
	api.PUT("/maintenance/:id", maintenance.Update)
	api.DELETE("/maintenance/:id", maintenance.Delete)
	hb := NewHeartbeatHandler(db)
	api.GET("/monitors/status", hb.GetRecentStatus)
	api.GET("/monitors/:id/beats", hb.GetBeats)
	api.GET("/monitors/:id/beats/important", hb.GetImportantBeats)
	api.GET("/monitors/:id/incidents", hb.GetIncidents)
	api.GET("/monitors/:id/status", hb.GetStatus)
	sla := NewSLAHandler(db)
	api.GET("/monitors/uptime/overall", sla.GetOverall)
	api.GET("/monitors/:id/uptime", sla.GetUptime)
	api.GET("/monitors/:id/uptime/data", sla.GetUptimeData)
	api.GET("/monitors/:id/uptime/summary", sla.GetUptimeSummary)
	return r, user
}

func TestUpdateUserProtectsLastAdminAndCanResetPassword(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	deactivateResp := authedRequest(t, r, http.MethodPatch, "/api/auth/users/1", gin.H{"active": false}, token)
	if deactivateResp.Code != http.StatusBadRequest {
		t.Fatalf("deactivate self code=%d body=%s", deactivateResp.Code, deactivateResp.Body.String())
	}

	role := model.RoleUser
	demoteResp := authedRequest(t, r, http.MethodPatch, "/api/auth/users/1", gin.H{"role": role}, token)
	if demoteResp.Code != http.StatusBadRequest {
		t.Fatalf("demote last admin code=%d body=%s", demoteResp.Code, demoteResp.Body.String())
	}

	otherPassword, _ := model.HashPassword("old-password")
	other := model.User{Username: "user", Password: otherPassword, Role: model.RoleUser, Active: true}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	resetResp := authedRequest(t, r, http.MethodPatch, "/api/auth/users/2", gin.H{"password": "new-password"}, token)
	if resetResp.Code != http.StatusOK {
		t.Fatalf("reset password code=%d body=%s", resetResp.Code, resetResp.Body.String())
	}
	var updated model.User
	db.First(&updated, other.ID)
	if !model.CheckPasswordHash("new-password", updated.Password) {
		t.Fatalf("password was not updated")
	}
}

func authedRequest(t *testing.T, r http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestInvalidMonitorIDReturnsBadRequest(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/monitors/bad"},
		{method: http.MethodPut, path: "/api/monitors/bad"},
		{method: http.MethodDelete, path: "/api/monitors/bad"},
		{method: http.MethodPost, path: "/api/monitors/bad/pause"},
		{method: http.MethodPost, path: "/api/monitors/bad/resume"},
		{method: http.MethodGet, path: "/api/monitors/bad/beats"},
		{method: http.MethodGet, path: "/api/monitors/bad/beats/important"},
		{method: http.MethodGet, path: "/api/monitors/bad/incidents"},
		{method: http.MethodGet, path: "/api/monitors/bad/status"},
		{method: http.MethodGet, path: "/api/monitors/bad/uptime"},
		{method: http.MethodGet, path: "/api/monitors/bad/uptime/data"},
		{method: http.MethodGet, path: "/api/monitors/bad/uptime/summary"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			resp := authedRequest(t, r, tt.method, tt.path, gin.H{"name": "ignored"}, token)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["code"] != "invalid_monitor_id" {
				t.Fatalf("body=%v", body)
			}
		})
	}
}

func TestInvalidResourceIDReturnsBadRequest(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	tests := []struct {
		name   string
		method string
		path   string
		code   string
	}{
		{name: "update user", method: http.MethodPatch, path: "/api/auth/users/bad", code: "invalid_user_id"},
		{name: "get notification", method: http.MethodGet, path: "/api/notifications/bad", code: "invalid_notification_id"},
		{name: "update notification", method: http.MethodPut, path: "/api/notifications/bad", code: "invalid_notification_id"},
		{name: "delete notification", method: http.MethodDelete, path: "/api/notifications/bad", code: "invalid_notification_id"},
		{name: "test notification", method: http.MethodPost, path: "/api/notifications/bad/test", code: "invalid_notification_id"},
		{name: "update maintenance", method: http.MethodPut, path: "/api/maintenance/bad", code: "invalid_maintenance_id"},
		{name: "delete maintenance", method: http.MethodDelete, path: "/api/maintenance/bad", code: "invalid_maintenance_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := authedRequest(t, r, tt.method, tt.path, gin.H{"name": "ignored"}, token)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["code"] != tt.code {
				t.Fatalf("body=%v want code %s", body, tt.code)
			}
		})
	}
}

func TestMonitorLookupDatabaseErrorReturnsServerError(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	if err := db.Migrator().DropTable(&model.Monitor{}); err != nil {
		t.Fatalf("drop monitors: %v", err)
	}

	resp := authedRequest(t, r, http.MethodGet, "/api/monitors/1", nil, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "monitor_lookup_failed" {
		t.Fatalf("body=%v", body)
	}
}

func TestCreateMonitorReturnsSchedulerStartErrors(t *testing.T) {
	db := testDB(t)
	schedulerErr := errors.New("scheduler unavailable")
	scheduler := &fakeScheduler{startErr: schedulerErr}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)
	notification := model.Notification{UserID: user.ID, Name: "ops", Type: model.NotificationTypeEmail, Config: `{"to":"ops@example.com"}`, Active: true}
	if err := db.Create(&notification).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}

	resp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":             "group",
		"type":             model.MonitorTypeGroup,
		"notification_ids": []uint{notification.ID},
		"tag_names":        []string{"prod"},
	}, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "monitor_scheduler_start_failed" {
		t.Fatalf("body=%v", body)
	}
	var monitorCount int64
	if err := db.Model(&model.Monitor{}).Where("user_id = ? AND name = ?", user.ID, "group").Count(&monitorCount).Error; err != nil {
		t.Fatalf("count monitors: %v", err)
	}
	if monitorCount != 0 {
		t.Fatalf("monitor count=%d want rollback to 0", monitorCount)
	}
	var notificationLinks int64
	if err := db.Model(&model.MonitorNotification{}).Count(&notificationLinks).Error; err != nil {
		t.Fatalf("count notification links: %v", err)
	}
	if notificationLinks != 0 {
		t.Fatalf("notification links=%d want rollback to 0", notificationLinks)
	}
	var tagLinks int64
	if err := db.Model(&model.MonitorTag{}).Count(&tagLinks).Error; err != nil {
		t.Fatalf("count tag links: %v", err)
	}
	if tagLinks != 0 {
		t.Fatalf("tag links=%d want rollback to 0", tagLinks)
	}
	var tagCount int64
	if err := db.Model(&model.Tag{}).Where("name = ?", "prod").Count(&tagCount).Error; err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if tagCount != 0 {
		t.Fatalf("tag count=%d want rollback to 0", tagCount)
	}
	wantCalls := []schedulerCall{
		{action: "start", id: 1},
		{action: "stop", id: 1},
	}
	assertSchedulerCalls(t, scheduler, wantCalls)
}

func TestCreateMonitorPreservesDisabledExpiryNotification(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	resp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":                "site",
		"type":                model.MonitorTypeHTTP,
		"url":                 "https://example.com",
		"expiry_notification": false,
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", resp.Code, resp.Body.String())
	}

	var created model.Monitor
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}
	if created.ExpiryNotification {
		t.Fatal("response should preserve expiry_notification=false")
	}
	var stored model.Monitor
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("load monitor: %v", err)
	}
	if stored.ExpiryNotification {
		t.Fatal("stored monitor should preserve expiry_notification=false")
	}
}

func TestPushMonitorMutationsDoNotStartScheduler(t *testing.T) {
	db := testDB(t)
	scheduler := &fakeScheduler{}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)

	createResp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name": "push",
		"type": model.MonitorTypePush,
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var created model.Monitor
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}
	if len(scheduler.calls) != 0 {
		t.Fatalf("push create should not start scheduler: %+v", scheduler.calls)
	}

	updateResp := authedRequest(t, r, http.MethodPut, fmt.Sprintf("/api/monitors/%d", created.ID), gin.H{
		"name": "push updated",
		"type": model.MonitorTypePush,
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	assertSchedulerCalls(t, scheduler, []schedulerCall{{action: "stop", id: created.ID}})
}

func TestCreateMonitorRollsBackDefaultCorrectionErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("monitor default correction failed")
	db.Callback().Update().Before("gorm:update").Register("test_fail_monitor_default_correction", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.Table, "monitors") {
			tx.AddError(wantErr)
		}
	})
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	resp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":                "site",
		"type":                model.MonitorTypeHTTP,
		"url":                 "https://example.com",
		"expiry_notification": false,
	}, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("create code=%d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "monitor_create_failed" {
		t.Fatalf("body=%v", body)
	}
	var monitorCount int64
	if err := db.Model(&model.Monitor{}).Where("name = ?", "site").Count(&monitorCount).Error; err != nil {
		t.Fatalf("count monitors: %v", err)
	}
	if monitorCount != 0 {
		t.Fatalf("monitor count=%d want rollback to 0", monitorCount)
	}
}

func TestCreateMonitorRollsBackMonitorWhenAssociationsFail(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	if err := db.Migrator().DropTable(&model.MonitorNotification{}); err != nil {
		t.Fatalf("drop monitor notifications: %v", err)
	}

	resp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":             "site",
		"type":             model.MonitorTypeHTTP,
		"url":              "https://example.com",
		"notification_ids": []uint{1},
	}, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("create code=%d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "monitor_create_failed" {
		t.Fatalf("body=%v", body)
	}
	var monitorCount int64
	if err := db.Model(&model.Monitor{}).Where("name = ?", "site").Count(&monitorCount).Error; err != nil {
		t.Fatalf("count monitors: %v", err)
	}
	if monitorCount != 0 {
		t.Fatalf("monitor count=%d want rollback to 0", monitorCount)
	}
}

func TestDeleteMissingOwnedResourcesReturnsNotFound(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	tests := []struct {
		name string
		path string
		code string
	}{
		{name: "notification", path: "/api/notifications/999", code: "notification_not_found"},
		{name: "maintenance", path: "/api/maintenance/999", code: "maintenance_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := authedRequest(t, r, http.MethodDelete, tt.path, nil, token)
			if resp.Code != http.StatusNotFound {
				t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["code"] != tt.code {
				t.Fatalf("body=%v want code %s", body, tt.code)
			}
		})
	}
}

func TestMonitorMutationsNotifyScheduler(t *testing.T) {
	db := testDB(t)
	scheduler := &fakeScheduler{}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)

	createResp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":     "site",
		"type":     "http",
		"url":      "https://example.com",
		"interval": 60,
		"timeout":  5,
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var created model.Monitor
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}

	updateResp := authedRequest(t, r, http.MethodPut, "/api/monitors/1", gin.H{
		"name":     "site2",
		"type":     "http",
		"url":      "https://example.org",
		"interval": 30,
		"timeout":  5,
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	pauseResp := authedRequest(t, r, http.MethodPost, "/api/monitors/1/pause", nil, token)
	if pauseResp.Code != http.StatusOK {
		t.Fatalf("pause code=%d body=%s", pauseResp.Code, pauseResp.Body.String())
	}
	resumeResp := authedRequest(t, r, http.MethodPost, "/api/monitors/1/resume", nil, token)
	if resumeResp.Code != http.StatusOK {
		t.Fatalf("resume code=%d body=%s", resumeResp.Code, resumeResp.Body.String())
	}
	deleteResp := authedRequest(t, r, http.MethodDelete, "/api/monitors/1", nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete code=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	assertSchedulerCalls(t, scheduler, []schedulerCall{
		{action: "start", id: 1},
		{action: "restart", id: 1},
		{action: "stop", id: 1},
		{action: "restart", id: 1},
		{action: "stop", id: 1},
	})
	_ = created
}

func TestUpdateMonitorDeactivatesOnSchedulerRestartError(t *testing.T) {
	db := testDB(t)
	schedulerErr := errors.New("restart unavailable")
	scheduler := &fakeScheduler{restartErr: schedulerErr}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)

	monitor := model.Monitor{
		UserID:              user.ID,
		Name:                "site",
		Type:                model.MonitorTypeHTTP,
		URL:                 "https://old.example.com",
		Active:              true,
		AcceptedStatusCodes: `["200-299"]`,
		ExpiryNotification:  true,
	}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	resp := authedRequest(t, r, http.MethodPut, fmt.Sprintf("/api/monitors/%d", monitor.ID), gin.H{
		"name":     "site",
		"type":     model.MonitorTypeHTTP,
		"url":      "https://new.example.com",
		"interval": 60,
		"timeout":  5,
	}, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("update code=%d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "monitor_scheduler_restart_failed" {
		t.Fatalf("body=%v", body)
	}

	var stored model.Monitor
	if err := db.First(&stored, monitor.ID).Error; err != nil {
		t.Fatalf("load monitor: %v", err)
	}
	if stored.Active {
		t.Fatal("monitor should be deactivated after scheduler restart failure")
	}
	if stored.URL != "https://new.example.com" {
		t.Fatalf("updated URL=%s want persisted update", stored.URL)
	}
	wantCalls := []schedulerCall{
		{action: "restart", id: monitor.ID},
		{action: "stop", id: monitor.ID},
	}
	assertSchedulerCalls(t, scheduler, wantCalls)
}

func TestImportCopyCountsOnceAndRefreshesOverwriteLinks(t *testing.T) {
	db := testDB(t)
	scheduler := &fakeScheduler{}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)

	monitor := model.Monitor{UserID: user.ID, Name: "site", Type: "http", URL: "https://old", Active: true, Interval: 60, Timeout: 5, AcceptedStatusCodes: `["200-299"]`}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	oldTag := model.Tag{Name: "old", Color: "#111111"}
	db.Create(&oldTag)
	db.Create(&model.MonitorTag{MonitorID: monitor.ID, TagID: oldTag.ID, Value: oldTag.Name})

	copyReq := ImportRequest{Strategy: "copy", Data: ExportFile{Monitors: []ExportMonitor{{
		Name: "site", Type: "http", URL: "https://copy", Active: true, Interval: 60, Timeout: 5, AcceptedStatusCodes: []string{"200-299"},
	}}}}
	copyResp := authedRequest(t, r, http.MethodPost, "/api/monitors/import", copyReq, token)
	if copyResp.Code != http.StatusOK {
		t.Fatalf("copy code=%d body=%s", copyResp.Code, copyResp.Body.String())
	}
	var copyResult ImportResult
	json.Unmarshal(copyResp.Body.Bytes(), &copyResult)
	if copyResult.Created != 1 || copyResult.Imported != 1 {
		t.Fatalf("copy result=%+v", copyResult)
	}

	overwriteReq := ImportRequest{Strategy: "overwrite", Data: ExportFile{Monitors: []ExportMonitor{{
		Name: "site", Type: "http", URL: "https://new", Active: true, Interval: 60, Timeout: 5, AcceptedStatusCodes: []string{"200-299"},
		Tags: []ExportTag{{Name: "new", Color: "#222222"}},
	}}}}
	overwriteResp := authedRequest(t, r, http.MethodPost, "/api/monitors/import", overwriteReq, token)
	if overwriteResp.Code != http.StatusOK {
		t.Fatalf("overwrite code=%d body=%s", overwriteResp.Code, overwriteResp.Body.String())
	}

	var tags []model.Tag
	db.Raw("SELECT t.* FROM tags t JOIN monitor_tags mt ON mt.tag_id = t.id WHERE mt.monitor_id = ?", monitor.ID).Scan(&tags)
	if len(tags) != 1 || tags[0].Name != "new" {
		t.Fatalf("tags=%+v", tags)
	}
}

func TestMaskSensitive(t *testing.T) {
	masked := maskSensitive(`{"webhook_url":"https://x","nested":{"password":"p"},"email":"ops@example.com"}`)
	if masked == "" || masked == `{"webhook_url":"https://x","nested":{"password":"p"},"email":"ops@example.com"}` {
		t.Fatalf("not masked: %s", masked)
	}
	if !containsMaskedValue(masked) {
		t.Fatalf("masked marker missing: %s", masked)
	}
}

func TestWebSocketTokenAuth(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	req := httptest.NewRequest(http.MethodGet, "/api/ws?token="+token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("valid token should reach websocket upgrade, code=%d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/ws?token=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestNotificationTestValidatesConfig(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	notif := model.Notification{UserID: user.ID, Name: "bad", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	db.Create(&notif)

	resp := authedRequest(t, r, http.MethodPost, "/api/notifications/1/test", nil, token)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("notification test code=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestNotificationDeleteRollsBackAssociationFailures(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	notif := model.Notification{UserID: user.ID, Name: "ops", Type: model.NotificationTypeFeishu, Config: `{}`, Active: true}
	if err := db.Create(&notif).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := db.Migrator().DropTable(&model.MonitorNotification{}); err != nil {
		t.Fatalf("drop monitor_notifications: %v", err)
	}

	resp := authedRequest(t, r, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", notif.ID), nil, token)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d body=%s", resp.Code, resp.Body.String())
	}

	var kept model.Notification
	if err := db.First(&kept, notif.ID).Error; err != nil {
		t.Fatalf("notification should be kept after rollback: %v", err)
	}
}

func TestOverallSLAPersistsReport(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	monitor := model.Monitor{UserID: user.ID, Name: "site", Type: "http", URL: "https://example.com", Active: true}
	db.Create(&monitor)
	now := time.Now()
	ping := 10.0
	db.Create(&model.Heartbeat{MonitorID: monitor.ID, Status: model.StatusUP, PingMS: &ping, Time: now.Add(-2 * time.Hour)})
	db.Create(&model.Heartbeat{MonitorID: monitor.ID, Status: model.StatusDown, Time: now.Add(-1 * time.Hour)})

	resp := authedRequest(t, r, http.MethodGet, "/api/monitors/uptime/overall?period=day", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("sla code=%d body=%s", resp.Code, resp.Body.String())
	}
	var reports int64
	db.Model(&model.SLAReport{}).Where("user_id = ?", user.ID).Count(&reports)
	if reports != 1 {
		t.Fatalf("reports=%d want 1", reports)
	}
}

func TestImportExportUserIsolation(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	other := model.User{Username: "other", Password: "hash", Role: model.RoleUser, Active: true}
	db.Create(&other)
	ownMonitor := model.Monitor{UserID: user.ID, Name: "own", Type: "http", URL: "https://own", Active: true, AcceptedStatusCodes: `["200-299"]`}
	otherMonitor := model.Monitor{UserID: other.ID, Name: "other", Type: "http", URL: "https://other", Active: true, AcceptedStatusCodes: `["200-299"]`}
	db.Create(&ownMonitor)
	db.Create(&otherMonitor)
	otherNotif := model.Notification{UserID: other.ID, Name: "ops", Type: model.NotificationTypeEmail, Config: `{"to":"other@example.com"}`, Active: true}
	ownNotif := model.Notification{UserID: user.ID, Name: "ops", Type: model.NotificationTypeEmail, Config: `{"to":"own@example.com"}`, Active: true}
	db.Create(&otherNotif)
	db.Create(&ownNotif)

	exportResp := authedRequest(t, r, http.MethodGet, "/api/monitors/export", nil, token)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("export code=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
	var file ExportFile
	json.Unmarshal(exportResp.Body.Bytes(), &file)
	if len(file.Monitors) != 1 || file.Monitors[0].Name != "own" {
		t.Fatalf("exported monitors=%+v", file.Monitors)
	}

	importResp := authedRequest(t, r, http.MethodPost, "/api/monitors/import", ImportRequest{
		Strategy: "overwrite",
		Data: ExportFile{Monitors: []ExportMonitor{
			{Name: "other", Type: "http", URL: "https://new", Active: true, AcceptedStatusCodes: []string{"200-299"}},
			{Name: "linked", Type: "http", URL: "https://linked", Active: true, AcceptedStatusCodes: []string{"200-299"}, NotificationNames: []string{"ops"}},
		}},
	}, token)
	if importResp.Code != http.StatusOK {
		t.Fatalf("import code=%d body=%s", importResp.Code, importResp.Body.String())
	}
	var stillOther model.Monitor
	db.First(&stillOther, otherMonitor.ID)
	if stillOther.URL != "https://other" {
		t.Fatalf("other user's monitor was modified: %s", stillOther.URL)
	}
	var linked model.Monitor
	if err := db.Where("user_id = ? AND name = ?", user.ID, "linked").First(&linked).Error; err != nil {
		t.Fatalf("linked monitor missing: %v", err)
	}
	var link model.MonitorNotification
	if err := db.Where("monitor_id = ?", linked.ID).First(&link).Error; err != nil {
		t.Fatalf("linked monitor notification missing: %v", err)
	}
	if link.NotificationID != ownNotif.ID {
		t.Fatalf("linked notification=%d want owned=%d", link.NotificationID, ownNotif.ID)
	}
}

func TestMonitorGroupValidationAndSchedulerStart(t *testing.T) {
	db := testDB(t)
	scheduler := &fakeScheduler{}
	r, user := setupRouter(t, db, scheduler)
	token := authToken(t, user)

	groupResp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name": "platform",
		"type": model.MonitorTypeGroup,
	}, token)
	if groupResp.Code != http.StatusCreated {
		t.Fatalf("group create code=%d body=%s", groupResp.Code, groupResp.Body.String())
	}
	var group model.Monitor
	if err := json.Unmarshal(groupResp.Body.Bytes(), &group); err != nil {
		t.Fatalf("decode group: %v", err)
	}
	assertSchedulerCalls(t, scheduler, []schedulerCall{{action: "start", id: group.ID}})

	childResp := authedRequest(t, r, http.MethodPost, "/api/monitors", gin.H{
		"name":     "site",
		"type":     model.MonitorTypeHTTP,
		"url":      "https://example.com",
		"group_id": group.ID,
	}, token)
	if childResp.Code != http.StatusCreated {
		t.Fatalf("child create code=%d body=%s", childResp.Code, childResp.Body.String())
	}

	invalidResp := authedRequest(t, r, http.MethodPut, "/api/monitors/1", gin.H{
		"name":     "platform",
		"type":     model.MonitorTypeGroup,
		"group_id": group.ID,
	}, token)
	if invalidResp.Code != http.StatusBadRequest {
		t.Fatalf("self group code=%d body=%s", invalidResp.Code, invalidResp.Body.String())
	}
}

func TestMonitorGroupStatusAggregatesChildren(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	root := model.Monitor{UserID: user.ID, Name: "root", Type: model.MonitorTypeGroup, Active: true}
	db.Create(&root)
	nested := model.Monitor{UserID: user.ID, Name: "nested", Type: model.MonitorTypeGroup, GroupID: &root.ID, Active: true}
	db.Create(&nested)
	leafUp := model.Monitor{UserID: user.ID, Name: "up", Type: model.MonitorTypeHTTP, GroupID: &nested.ID, Active: true}
	leafDown := model.Monitor{UserID: user.ID, Name: "down", Type: model.MonitorTypeHTTP, GroupID: &root.ID, Active: true}
	db.Create(&leafUp)
	db.Create(&leafDown)
	ping := 10.0
	db.Create(&model.Heartbeat{MonitorID: leafUp.ID, Status: model.StatusUP, PingMS: &ping, Time: time.Now().Add(-10 * time.Minute)})
	db.Create(&model.Heartbeat{MonitorID: leafDown.ID, Status: model.StatusDown, Time: time.Now().Add(-5 * time.Minute)})

	resp := authedRequest(t, r, http.MethodGet, "/api/monitors/status", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("status code=%d body=%s", resp.Code, resp.Body.String())
	}
	var statuses []engine.MonitorStatusSnapshot
	if err := json.Unmarshal(resp.Body.Bytes(), &statuses); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	byID := map[uint]engine.MonitorStatusSnapshot{}
	for _, s := range statuses {
		byID[s.ID] = s
	}
	if byID[root.ID].Status != model.StatusDown {
		t.Fatalf("root status=%d want DOWN", byID[root.ID].Status)
	}
	if byID[nested.ID].Status != model.StatusUP {
		t.Fatalf("nested status=%d want UP", byID[nested.ID].Status)
	}
	if byID[leafUp.ID].GroupID == nil || *byID[leafUp.ID].GroupID != nested.ID {
		t.Fatalf("leaf group_id=%v want %d", byID[leafUp.ID].GroupID, nested.ID)
	}
}

func TestImportExportPreservesGroupPath(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)

	root := model.Monitor{UserID: user.ID, Name: "root", Type: model.MonitorTypeGroup, Active: true}
	db.Create(&root)
	child := model.Monitor{UserID: user.ID, Name: "child", Type: model.MonitorTypeGroup, GroupID: &root.ID, Active: true}
	db.Create(&child)
	leaf := model.Monitor{UserID: user.ID, Name: "site", Type: model.MonitorTypeHTTP, URL: "https://example.com", GroupID: &child.ID, Active: true, AcceptedStatusCodes: `["200-299"]`}
	db.Create(&leaf)

	exportResp := authedRequest(t, r, http.MethodGet, "/api/monitors/export", nil, token)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("export code=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
	var file ExportFile
	if err := json.Unmarshal(exportResp.Body.Bytes(), &file); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	var exportedLeaf ExportMonitor
	for _, m := range file.Monitors {
		if m.Name == "site" {
			exportedLeaf = m
		}
	}
	if strings.Join(exportedLeaf.GroupPath, "/") != "root/child" {
		t.Fatalf("group path=%v", exportedLeaf.GroupPath)
	}

	db2 := testDB(t)
	r2, user2 := setupRouter(t, db2, &fakeScheduler{})
	token2 := authToken(t, user2)
	importResp := authedRequest(t, r2, http.MethodPost, "/api/monitors/import", ImportRequest{Strategy: "copy", Data: ExportFile{Monitors: []ExportMonitor{exportedLeaf}}}, token2)
	if importResp.Code != http.StatusOK {
		t.Fatalf("import code=%d body=%s", importResp.Code, importResp.Body.String())
	}
	var imported model.Monitor
	if err := db2.Where("user_id = ? AND name = ?", user2.ID, "site").First(&imported).Error; err != nil {
		t.Fatalf("imported leaf missing: %v", err)
	}
	if imported.GroupID == nil {
		t.Fatalf("imported leaf missing group")
	}
	var importedGroup model.Monitor
	db2.First(&importedGroup, *imported.GroupID)
	if importedGroup.Name != "child" {
		t.Fatalf("imported group=%+v", importedGroup)
	}
}

func TestMaintenanceWindowCRUD(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	monitor := model.Monitor{UserID: user.ID, Name: "site", Type: model.MonitorTypeHTTP, URL: "https://example.com", Active: true}
	db.Create(&monitor)

	createResp := authedRequest(t, r, http.MethodPost, "/api/maintenance", gin.H{
		"name":       "deploy",
		"monitor_id": monitor.ID,
		"start_at":   time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"end_at":     time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		"active":     true,
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var window model.MaintenanceWindow
	if err := json.Unmarshal(createResp.Body.Bytes(), &window); err != nil {
		t.Fatalf("decode window: %v", err)
	}
	if window.MonitorID == nil || *window.MonitorID != monitor.ID {
		t.Fatalf("monitor_id=%v want %d", window.MonitorID, monitor.ID)
	}

	updateResp := authedRequest(t, r, http.MethodPut, fmt.Sprintf("/api/maintenance/%d", window.ID), gin.H{
		"name":     "deploy updated",
		"start_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"end_at":   time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"active":   false,
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	listResp := authedRequest(t, r, http.MethodGet, "/api/maintenance", nil, token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list code=%d body=%s", listResp.Code, listResp.Body.String())
	}
	deleteResp := authedRequest(t, r, http.MethodDelete, fmt.Sprintf("/api/maintenance/%d", window.ID), nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete code=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestCreateMaintenanceWindowPreservesInactive(t *testing.T) {
	db := testDB(t)
	r, user := setupRouter(t, db, &fakeScheduler{})
	token := authToken(t, user)
	start := time.Now().Add(-time.Hour).UTC()

	resp := authedRequest(t, r, http.MethodPost, "/api/maintenance", gin.H{
		"name":     "disabled",
		"start_at": start.Format(time.RFC3339),
		"end_at":   start.Add(time.Hour).Format(time.RFC3339),
		"active":   false,
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", resp.Code, resp.Body.String())
	}

	var window model.MaintenanceWindow
	if err := json.Unmarshal(resp.Body.Bytes(), &window); err != nil {
		t.Fatalf("decode window: %v", err)
	}
	if window.Active {
		t.Fatal("response should preserve active=false")
	}
	var stored model.MaintenanceWindow
	if err := db.First(&stored, window.ID).Error; err != nil {
		t.Fatalf("load window: %v", err)
	}
	if stored.Active {
		t.Fatal("stored window should preserve active=false")
	}
}
