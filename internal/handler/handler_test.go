package handler

import (
	"bytes"
	"encoding/json"
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
	calls []schedulerCall
}

var testDBSeq uint64

func (s *fakeScheduler) StartMonitor(m *model.Monitor) {
	s.calls = append(s.calls, schedulerCall{action: "start", id: m.ID})
}

func (s *fakeScheduler) StopMonitor(id uint) {
	s.calls = append(s.calls, schedulerCall{action: "stop", id: id})
}

func (s *fakeScheduler) RestartMonitor(m *model.Monitor) {
	s.calls = append(s.calls, schedulerCall{action: "restart", id: m.ID})
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
	api.PUT("/monitors/:id", monitor.Update)
	api.DELETE("/monitors/:id", monitor.Delete)
	api.POST("/monitors/:id/resume", monitor.Resume)
	api.POST("/monitors/:id/pause", monitor.Pause)
	api.PATCH("/auth/users/:id", middleware.AdminRequired(), auth.UpdateUser)
	ie := NewImportExportHandler(db, scheduler)
	api.GET("/monitors/export", ie.ExportMonitors)
	api.POST("/monitors/import", ie.ImportExecute)
	notif := NewNotificationHandler(db)
	api.POST("/notifications/:id/test", notif.Test)
	maintenance := NewMaintenanceHandler(db)
	api.GET("/maintenance", maintenance.List)
	api.POST("/maintenance", maintenance.Create)
	api.PUT("/maintenance/:id", maintenance.Update)
	api.DELETE("/maintenance/:id", maintenance.Delete)
	hb := NewHeartbeatHandler(db)
	api.GET("/monitors/status", hb.GetRecentStatus)
	api.GET("/monitors/:id/status", hb.GetStatus)
	sla := NewSLAHandler(db)
	api.GET("/monitors/uptime/overall", sla.GetOverall)
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

	got := []string{}
	for _, call := range scheduler.calls {
		got = append(got, call.action)
	}
	want := []string{"start", "restart", "stop", "restart", "stop"}
	if len(got) != len(want) {
		t.Fatalf("scheduler calls=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scheduler calls=%v want=%v", got, want)
		}
	}
	_ = created
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
	notif := model.Notification{UserID: user.ID, Name: "bad", Type: "feishu", Config: `{}`, Active: true}
	db.Create(&notif)

	resp := authedRequest(t, r, http.MethodPost, "/api/notifications/1/test", nil, token)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("notification test code=%d body=%s", resp.Code, resp.Body.String())
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
		Data: ExportFile{Monitors: []ExportMonitor{{
			Name: "other", Type: "http", URL: "https://new", Active: true, AcceptedStatusCodes: []string{"200-299"},
		}}},
	}, token)
	if importResp.Code != http.StatusOK {
		t.Fatalf("import code=%d body=%s", importResp.Code, importResp.Body.String())
	}
	var stillOther model.Monitor
	db.First(&stillOther, otherMonitor.ID)
	if stillOther.URL != "https://other" {
		t.Fatalf("other user's monitor was modified: %s", stillOther.URL)
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
	if len(scheduler.calls) != 1 || scheduler.calls[0].action != "start" {
		t.Fatalf("group should start scheduler: %+v", scheduler.calls)
	}

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
