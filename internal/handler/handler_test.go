package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/config"
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
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
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
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	monitor := NewMonitorHandler(db, scheduler)
	api.POST("/monitors", monitor.Create)
	api.PUT("/monitors/:id", monitor.Update)
	api.DELETE("/monitors/:id", monitor.Delete)
	api.POST("/monitors/:id/resume", monitor.Resume)
	api.POST("/monitors/:id/pause", monitor.Pause)
	ie := NewImportExportHandler(db, scheduler)
	api.POST("/monitors/import", ie.ImportExecute)
	return r, user
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
	want := []string{"start", "restart", "stop", "start", "stop"}
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
