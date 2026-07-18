package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

var schedulerTestDBSeq uint64

type fakeChecker struct {
	result *CheckResult
	err    error
}

func (c fakeChecker) Check(*model.Monitor) (*CheckResult, error) {
	return c.result, c.err
}

func TestSchedulerRunsGroupMonitors(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartMonitor(&group); err != nil {
		t.Fatalf("start monitor: %v", err)
	}
	defer scheduler.StopAll()
	if scheduler.RunningCount() != 1 {
		t.Fatalf("running=%d want 1", scheduler.RunningCount())
	}
}

func TestSchedulerUsesInjectedCheckerForInitialBeat(t *testing.T) {
	db := schedulerTestDB(t)
	monitor := model.Monitor{UserID: 1, Name: "site", Type: model.MonitorTypeHTTP, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&monitor).Error; err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	scheduler.checkerProvider = func(monitorType string) Checker {
		if monitorType != model.MonitorTypeHTTP {
			t.Fatalf("monitorType=%s", monitorType)
		}
		return fakeChecker{result: &CheckResult{Status: model.StatusUP, PingMS: 12, Msg: "ok"}}
	}
	if err := scheduler.StartMonitor(&monitor); err != nil {
		t.Fatalf("start monitor: %v", err)
	}
	defer scheduler.StopAll()

	var beat model.Heartbeat
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := db.Where("monitor_id = ?", monitor.ID).First(&beat).Error; err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if beat.ID == 0 {
		t.Fatal("initial heartbeat missing")
	}
	if beat.Status != model.StatusUP || beat.Msg != "ok" || beat.PingMS == nil || *beat.PingMS != 12 {
		t.Fatalf("beat=%+v", beat)
	}
}

func TestSchedulerSkipsPushMonitors(t *testing.T) {
	db := schedulerTestDB(t)
	push := model.Monitor{UserID: 1, Name: "push", Type: model.MonitorTypePush, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&push).Error; err != nil {
		t.Fatalf("create push monitor: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartMonitor(&push); err != nil {
		t.Fatalf("start push monitor: %v", err)
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
	if err := scheduler.StartAll(); err != nil {
		t.Fatalf("start all: %v", err)
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running after start all=%d want 0", scheduler.RunningCount())
	}
}

func TestSchedulerStopMonitorWaitsForRunnerExit(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartMonitor(&group); err != nil {
		t.Fatalf("start monitor: %v", err)
	}
	runner := scheduler.monitors[group.ID]

	scheduler.StopMonitor(group.ID)

	select {
	case <-runner.DoneChan:
	default:
		t.Fatal("runner should be stopped when StopMonitor returns")
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
	if err := scheduler.StartMonitor(&group); err != nil {
		t.Fatalf("restart monitor: %v", err)
	}
	defer scheduler.StopAll()
	if scheduler.RunningCount() != 1 {
		t.Fatalf("running after restart=%d want 1", scheduler.RunningCount())
	}
}

func TestSchedulerStopAllWaitsForRunnerExit(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartMonitor(&group); err != nil {
		t.Fatalf("start monitor: %v", err)
	}
	runner := scheduler.monitors[group.ID]

	scheduler.StopAll()

	select {
	case <-runner.DoneChan:
	default:
		t.Fatal("runner should be stopped when StopAll returns")
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
}

func TestSchedulerStopAllPreventsFutureStarts(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	scheduler := NewScheduler(db, nil)
	scheduler.StopAll()

	if err := scheduler.StartMonitor(&group); !errors.Is(err, errSchedulerStopped) {
		t.Fatalf("start error=%v want scheduler stopped", err)
	}
	if err := scheduler.RestartMonitor(&group); !errors.Is(err, errSchedulerStopped) {
		t.Fatalf("restart error=%v want scheduler stopped", err)
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
}

func TestSchedulerStartMonitorReturnsInitErrors(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := db.Migrator().DropTable(&model.StatMinutely{}); err != nil {
		t.Fatalf("drop stat minutely: %v", err)
	}

	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartMonitor(&group); err == nil {
		t.Fatal("expected start error")
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
}

func TestSchedulerStartAllReturnsMonitorStartErrors(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := db.Migrator().DropTable(&model.StatMinutely{}); err != nil {
		t.Fatalf("drop stat minutely: %v", err)
	}

	scheduler := NewScheduler(db, nil)
	if err := scheduler.StartAll(); err == nil {
		t.Fatal("expected start all error")
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want 0", scheduler.RunningCount())
	}
}

func TestSchedulerStartAllRollsBackStartedMonitors(t *testing.T) {
	db := schedulerTestDB(t)
	first := model.Monitor{UserID: 1, Name: "first", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	second := model.Monitor{UserID: 1, Name: "second", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second: %v", err)
	}
	wantErr := errors.New("stat init failed")
	db.Callback().Query().After("gorm:query").Register("test_fail_second_stat_init", func(tx *gorm.DB) {
		if !strings.Contains(tx.Statement.SQL.String(), "stat_minutelies") {
			return
		}
		for _, arg := range tx.Statement.Vars {
			if fmt.Sprint(arg) == fmt.Sprint(second.ID) {
				tx.AddError(wantErr)
			}
		}
	})

	scheduler := NewScheduler(db, nil)
	err := scheduler.StartAll()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}
	if scheduler.RunningCount() != 0 {
		t.Fatalf("running=%d want rollback to 0", scheduler.RunningCount())
	}
	if _, exists := scheduler.monitors[first.ID]; exists {
		t.Fatal("first monitor should have been rolled back")
	}
	if _, exists := scheduler.monitors[second.ID]; exists {
		t.Fatal("second monitor should not be running")
	}
	var heartbeatCount int64
	if err := db.Model(&model.Heartbeat{}).Where("monitor_id = ?", first.ID).Count(&heartbeatCount).Error; err != nil {
		t.Fatalf("count first heartbeats: %v", err)
	}
	if heartbeatCount != 0 {
		t.Fatalf("first heartbeat count=%d want 0", heartbeatCount)
	}
}

func TestGroupBeatPersistsMaintenanceStatus(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	db.Create(&group)
	db.Create(&model.MaintenanceWindow{
		UserID:  1,
		Name:    "planned",
		StartAt: time.Now().Add(-time.Hour),
		EndAt:   time.Now().Add(time.Hour),
		Active:  true,
	})
	runner := &MonitorRunner{Monitor: &group, Calculator: NewUptimeCalculator(group.ID, db), DB: db}
	if err := runner.Calculator.Init(); err != nil {
		t.Fatalf("calc init: %v", err)
	}
	runner.beat()
	var beat model.Heartbeat
	if err := db.Where("monitor_id = ?", group.ID).First(&beat).Error; err != nil {
		t.Fatalf("heartbeat missing: %v", err)
	}
	if beat.Status != model.StatusMaintenance {
		t.Fatalf("status=%d want maintenance", beat.Status)
	}
}

func TestGroupBeatPersistsAggregatedStatus(t *testing.T) {
	db := schedulerTestDB(t)
	group := model.Monitor{UserID: 1, Name: "group", Type: model.MonitorTypeGroup, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	child := model.Monitor{UserID: 1, Name: "child", Type: model.MonitorTypeHTTP, GroupID: &group.ID, Active: true, Interval: model.DefaultInterval}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := db.Create(&model.Heartbeat{MonitorID: child.ID, Status: model.StatusUP, Time: time.Now()}).Error; err != nil {
		t.Fatalf("create child heartbeat: %v", err)
	}

	runner := &MonitorRunner{Monitor: &group, Calculator: NewUptimeCalculator(group.ID, db), DB: db}
	if err := runner.Calculator.Init(); err != nil {
		t.Fatalf("calc init: %v", err)
	}
	runner.beat()

	var beat model.Heartbeat
	if err := db.Where("monitor_id = ?", group.ID).Order("time DESC").First(&beat).Error; err != nil {
		t.Fatalf("group heartbeat missing: %v", err)
	}
	if beat.Status != model.StatusUP {
		t.Fatalf("status=%d want UP", beat.Status)
	}
}

func schedulerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	seq := atomic.AddUint64(&schedulerTestDBSeq, 1)
	name := fmt.Sprintf("%s_%d", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()), seq)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Monitor{},
		&model.Heartbeat{},
		&model.StatMinutely{},
		&model.StatHourly{},
		&model.StatDaily{},
		&model.MaintenanceWindow{},
		&model.Notification{},
		&model.MonitorNotification{},
		&model.Incident{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
