package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/config"
	"uptime_ng/internal/handler"
	"uptime_ng/internal/model"
)

func TestAPIDocsMentionRegisteredResourceGroups(t *testing.T) {
	content, err := os.ReadFile("../../docs/API.md")
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	doc := string(content)
	required := []string{
		"/auth/register",
		"/monitors",
		"/monitors/:id/status",
		"/notifications",
		"/maintenance",
		"/monitors/export",
		"/api/ws",
	}
	for _, item := range required {
		if !strings.Contains(doc, item) {
			t.Fatalf("docs/API.md missing %s", item)
		}
	}
}

func TestMonitorStatusRoutesUseConcreteAndIDPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHours: 72}}
	db, err := gorm.Open(sqlite.Open("file:router_status?mode=memory&cache=shared"), &gorm.Config{})
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
		t.Fatalf("migrate: %v", err)
	}
	user := model.User{Username: "admin", Password: "hash", Role: model.RoleAdmin, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	monitor := model.Monitor{UserID: user.ID, Name: "site", Type: model.MonitorTypeHTTP, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if err := db.Create(&model.Heartbeat{MonitorID: monitor.ID, Status: model.StatusUP, Time: time.Now()}).Error; err != nil {
		t.Fatalf("create beat: %v", err)
	}
	token, err := model.GenerateJWT(&user, config.AppConfig.JWT.Secret, 72)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}

	r := gin.New()
	Setup(r, db, handler.NewWSHub(), nil)

	listResp := authedGET(r, "/api/monitors/status", token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status code=%d body=%s", listResp.Code, listResp.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode status list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("status list len=%d want 1", len(list))
	}

	itemResp := authedGET(r, "/api/monitors/1/status", token)
	if itemResp.Code != http.StatusOK {
		t.Fatalf("item status code=%d body=%s", itemResp.Code, itemResp.Body.String())
	}
	var item map[string]any
	if err := json.Unmarshal(itemResp.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode status item: %v", err)
	}
	if item["name"] != "site" {
		t.Fatalf("item name=%v want site", item["name"])
	}
}

func authedGET(r http.Handler, path string, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
