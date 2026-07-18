package handler

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestImportNotificationsKeepsMaskedExistingSecret(t *testing.T) {
	db := testDB(t)
	userID := uint(1)
	existing := model.Notification{
		UserID: userID,
		Name:   "ops",
		Type:   model.NotificationTypeFeishu,
		Config: `{"webhook_url":"https://real"}`,
		Active: true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing: %v", err)
	}

	tx := db.Begin()
	if err := importNotifications(tx, userID, []ExportNotification{{
		Name:   "ops",
		Type:   model.NotificationTypeEmail,
		Config: `{"webhook_url":"***"}`,
	}}, "overwrite"); err != nil {
		t.Fatalf("importNotifications: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit: %v", err)
	}

	var got model.Notification
	if err := db.First(&got, existing.ID).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if got.Type != model.NotificationTypeEmail {
		t.Fatalf("type=%s want email", got.Type)
	}
	if got.Config != existing.Config {
		t.Fatalf("masked config overwrote secret: %s", got.Config)
	}
}

func TestImportNotificationsCreatesUnmaskedNotifications(t *testing.T) {
	db := testDB(t)

	err := runTransaction(db, func(tx *gorm.DB) error {
		return importNotifications(tx, 1, []ExportNotification{{
			Name:   "ops",
			Type:   model.NotificationTypeEmail,
			Config: `not-json-but-allowed-here`,
		}}, "copy")
	})
	if err != nil {
		t.Fatalf("importNotifications valid create: %v", err)
	}

	var count int64
	db.Model(&model.Notification{}).Where("name = ?", "ops").Count(&count)
	if count != 1 {
		t.Fatalf("notification count=%d want 1", count)
	}
}

func TestImportNotificationsReturnsDatabaseErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("db unavailable")
	tx := db.Session(&gorm.Session{DryRun: true})
	tx.AddError(wantErr)

	err := importNotifications(tx, 1, []ExportNotification{{
		Name:   "ops",
		Type:   model.NotificationTypeEmail,
		Config: `{}`,
	}}, "copy")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestImportMonitorReturnsLookupErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("lookup failed")
	tx := db.Session(&gorm.Session{DryRun: true})
	tx.AddError(wantErr)

	_, err := importMonitor(tx, 1, ExportMonitor{
		Name:                "site",
		Type:                model.MonitorTypeHTTP,
		AcceptedStatusCodes: []string{"200-299"},
	}, "skip")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestImportMonitorAppliesStrategies(t *testing.T) {
	tests := []struct {
		name         string
		strategy     string
		wantAction   importMonitorAction
		wantName     string
		wantExisting string
	}{
		{name: "skip existing", strategy: "skip", wantAction: importMonitorSkipped, wantName: "site", wantExisting: "https://old"},
		{name: "overwrite existing", strategy: "overwrite", wantAction: importMonitorUpdated, wantName: "site", wantExisting: "https://new"},
		{name: "copy existing", strategy: "copy", wantAction: importMonitorCreated, wantName: "site (copy)", wantExisting: "https://old"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB(t)
			existing := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, URL: "https://old"}
			if err := db.Create(&existing).Error; err != nil {
				t.Fatalf("create existing monitor: %v", err)
			}

			var outcome importMonitorOutcome
			if err := runTransaction(db, func(tx *gorm.DB) error {
				var err error
				outcome, err = importMonitor(tx, 1, ExportMonitor{
					Name:                "site",
					Type:                model.MonitorTypeHTTP,
					URL:                 "https://new",
					AcceptedStatusCodes: []string{"200-299"},
				}, tt.strategy)
				return err
			}); err != nil {
				t.Fatalf("importMonitor: %v", err)
			}

			if outcome.action != tt.wantAction {
				t.Fatalf("action=%s want %s", outcome.action, tt.wantAction)
			}
			if outcome.action != importMonitorSkipped && outcome.monitor.Name != tt.wantName {
				t.Fatalf("monitor name=%s want %s", outcome.monitor.Name, tt.wantName)
			}

			var gotExisting model.Monitor
			if err := db.First(&gotExisting, existing.ID).Error; err != nil {
				t.Fatalf("load existing monitor: %v", err)
			}
			if gotExisting.URL != tt.wantExisting {
				t.Fatalf("existing URL=%s want %s", gotExisting.URL, tt.wantExisting)
			}
		})
	}
}

func TestImportMonitorCreatesNewMonitorWithGroupPath(t *testing.T) {
	db := testDB(t)

	var outcome importMonitorOutcome
	if err := runTransaction(db, func(tx *gorm.DB) error {
		var err error
		outcome, err = importMonitor(tx, 1, ExportMonitor{
			Name:                "site",
			Type:                model.MonitorTypeHTTP,
			URL:                 "https://new",
			GroupPath:           []string{"root", "child"},
			AcceptedStatusCodes: []string{"200-299"},
		}, "skip")
		return err
	}); err != nil {
		t.Fatalf("importMonitor: %v", err)
	}

	if outcome.action != importMonitorCreated {
		t.Fatalf("action=%s want %s", outcome.action, importMonitorCreated)
	}
	var group model.Monitor
	if err := db.Where("user_id = ? AND name = ?", uint(1), "child").First(&group).Error; err != nil {
		t.Fatalf("child group missing: %v", err)
	}
	if outcome.monitor.GroupID == nil || *outcome.monitor.GroupID != group.ID {
		t.Fatalf("monitor group_id=%v want %d", outcome.monitor.GroupID, group.ID)
	}
}

func TestImportMonitorPreservesInactiveNewMonitor(t *testing.T) {
	db := testDB(t)

	var outcome importMonitorOutcome
	if err := runTransaction(db, func(tx *gorm.DB) error {
		var err error
		outcome, err = importMonitor(tx, 1, ExportMonitor{
			Name:                "paused",
			Type:                model.MonitorTypeHTTP,
			URL:                 "https://paused",
			Active:              false,
			AcceptedStatusCodes: []string{"200-299"},
		}, "skip")
		return err
	}); err != nil {
		t.Fatalf("importMonitor: %v", err)
	}

	if outcome.monitor.Active {
		t.Fatal("outcome monitor should preserve active=false")
	}
	var stored model.Monitor
	if err := db.First(&stored, outcome.monitor.ID).Error; err != nil {
		t.Fatalf("load imported monitor: %v", err)
	}
	if stored.Active {
		t.Fatal("stored monitor should preserve active=false")
	}
}

func TestImportMonitorPreservesDisabledExpiryNotification(t *testing.T) {
	db := testDB(t)

	var outcome importMonitorOutcome
	if err := runTransaction(db, func(tx *gorm.DB) error {
		var err error
		outcome, err = importMonitor(tx, 1, ExportMonitor{
			Name:                "site",
			Type:                model.MonitorTypeHTTP,
			URL:                 "https://site",
			Active:              true,
			ExpiryNotification:  false,
			AcceptedStatusCodes: []string{"200-299"},
		}, "skip")
		return err
	}); err != nil {
		t.Fatalf("importMonitor: %v", err)
	}

	if outcome.monitor.ExpiryNotification {
		t.Fatal("outcome monitor should preserve expiry_notification=false")
	}
	var stored model.Monitor
	if err := db.First(&stored, outcome.monitor.ID).Error; err != nil {
		t.Fatalf("load imported monitor: %v", err)
	}
	if stored.ExpiryNotification {
		t.Fatal("stored monitor should preserve expiry_notification=false")
	}
}

func TestEnsureGroupPathReturnsLookupErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("lookup failed")
	tx := db.Session(&gorm.Session{DryRun: true})
	tx.AddError(wantErr)

	_, err := ensureGroupPath(tx, 1, []string{"root"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestSyncImportedMonitorSchedulers(t *testing.T) {
	db := testDB(t)
	scheduler := &fakeScheduler{}
	if err := syncImportedMonitorSchedulers(db, scheduler, []model.Monitor{
		{ID: 1, Type: model.MonitorTypeGroup, Active: true},
		{ID: 2, Type: model.MonitorTypeHTTP, Active: true},
		{ID: 3, Type: model.MonitorTypeTCP, Active: false},
		{ID: 4, Type: model.MonitorTypePush, Active: true},
	}); err != nil {
		t.Fatalf("sync schedulers: %v", err)
	}

	want := []schedulerCall{
		{action: "restart", id: 1},
		{action: "restart", id: 2},
		{action: "stop", id: 3},
		{action: "stop", id: 4},
	}
	assertSchedulerCalls(t, scheduler, want)
}

func TestSyncImportedMonitorSchedulersReturnsSchedulerErrors(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("restart failed")
	scheduler := &fakeScheduler{restartErr: wantErr}
	err := syncImportedMonitorSchedulers(db, scheduler, []model.Monitor{{ID: 2, Type: model.MonitorTypeHTTP, Active: true}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
}

func TestSyncImportedMonitorSchedulersDeactivatesFailedRestart(t *testing.T) {
	db := testDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	wantErr := errors.New("restart failed")
	scheduler := &fakeScheduler{restartErr: wantErr}
	err := syncImportedMonitorSchedulers(db, scheduler, []model.Monitor{monitor})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}

	var stored model.Monitor
	if err := db.First(&stored, monitor.ID).Error; err != nil {
		t.Fatalf("load monitor: %v", err)
	}
	if stored.Active {
		t.Fatal("monitor should be deactivated after imported scheduler restart failure")
	}
	wantCalls := []schedulerCall{
		{action: "restart", id: monitor.ID},
		{action: "stop", id: monitor.ID},
	}
	assertSchedulerCalls(t, scheduler, wantCalls)
}
