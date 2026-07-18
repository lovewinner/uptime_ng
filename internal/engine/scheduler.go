package engine

import (
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type HeartbeatPublisher interface {
	SendToUser(userID uint, msgType string, payload interface{})
}

type Scheduler struct {
	DB        *gorm.DB
	publisher HeartbeatPublisher
	monitors  map[uint]*MonitorRunner
	mu        sync.RWMutex
	running   bool
}

type MonitorRunner struct {
	Monitor    *model.Monitor
	Ticker     *time.Ticker
	StopChan   chan struct{}
	Calculator *UptimeCalculator
	DB         *gorm.DB
	Publisher  HeartbeatPublisher
}

func NewScheduler(db *gorm.DB, publisher HeartbeatPublisher) *Scheduler {
	return &Scheduler{
		DB:        db,
		publisher: publisher,
		monitors:  make(map[uint]*MonitorRunner),
		running:   true,
	}
}

func (s *Scheduler) StartAll() error {
	var monitors []model.Monitor
	if err := s.DB.Where("active = ?", true).Find(&monitors).Error; err != nil {
		return err
	}

	log.Printf("Scheduler starting %d monitors", len(monitors))

	for i := range monitors {
		s.StartMonitor(&monitors[i])
	}

	return nil
}

func (s *Scheduler) StartMonitor(monitor *model.Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.monitors[monitor.ID]; exists {
		return
	}

	interval := monitor.Interval
	if interval < model.MinInterval {
		interval = model.DefaultInterval
	}

	calc := NewUptimeCalculator(monitor.ID, s.DB)
	if err := calc.Init(); err != nil {
		log.Printf("Failed to init uptime calc for monitor %d: %v", monitor.ID, err)
	}

	runner := &MonitorRunner{
		Monitor:    monitor,
		Ticker:     time.NewTicker(time.Duration(interval) * time.Second),
		StopChan:   make(chan struct{}),
		Calculator: calc,
		DB:         s.DB,
		Publisher:  s.publisher,
	}

	s.monitors[monitor.ID] = runner

	go func() {
		runner.run()
	}()

	log.Printf("Started monitor: %s (id=%d, interval=%ds)", monitor.Name, monitor.ID, interval)
}

func (s *Scheduler) StopMonitor(monitorID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, exists := s.monitors[monitorID]
	if !exists {
		return
	}

	close(runner.StopChan)
	runner.Ticker.Stop()
	delete(s.monitors, monitorID)
}

func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	for id, runner := range s.monitors {
		close(runner.StopChan)
		runner.Ticker.Stop()
		delete(s.monitors, id)
	}
}

func (s *Scheduler) RestartMonitor(monitor *model.Monitor) {
	s.StopMonitor(monitor.ID)
	time.Sleep(100 * time.Millisecond)
	s.StartMonitor(monitor)
}

func (s *Scheduler) RunningCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.monitors)
}

func (r *MonitorRunner) run() {
	if r.Monitor.Type == model.MonitorTypePush {
		return
	}

	log.Printf("Beat: %s", r.Monitor.Name)

	r.beat()

	for {
		select {
		case <-r.Ticker.C:
			r.beat()
		case <-r.StopChan:
			return
		}
	}
}

func (r *MonitorRunner) beat() {
	monitor := r.Monitor
	if window, ok := r.activeMaintenanceWindow(monitor, time.Now()); ok {
		r.maintenanceBeat(window)
		return
	}
	if monitor.Type == model.MonitorTypeGroup {
		r.groupBeat()
		return
	}
	checker := GetChecker(monitor.Type)
	if checker == nil {
		log.Printf("Unknown monitor type: %s for monitor %d", monitor.Type, monitor.ID)
		return
	}

	result, err := checker.Check(monitor)
	if err != nil {
		log.Printf("Check error for monitor %s: %v", monitor.Name, err)
		return
	}

	if monitor.UpsideDown {
		result.Status = model.FlipStatus(result.Status)
	}
	if monitor.Type == model.MonitorTypeDNS {
		r.DB.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("dns_last_result", result.Msg)
		monitor.DNSLastResult = result.Msg
	}

	now := time.Now()

	beat := model.Heartbeat{
		MonitorID:  monitor.ID,
		Status:     result.Status,
		Msg:        result.Msg,
		PingMS:     &result.PingMS,
		HTTPStatus: result.HTTPStatus,
		Time:       now,
	}

	var previousBeat model.Heartbeat
	r.DB.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&previousBeat)

	transition := applyCheckHeartbeatState(&beat, previousBeat, monitor, result)
	if beat.Important {
		r.sendNotification(transition.isFirstBeat, transition.previousStatus, beat)
	} else if beat.Status == model.StatusDown && monitor.ResendInterval > 0 {
		if r.shouldResendDownNotification(now, monitor.ResendInterval) {
			beat.Important = true
			r.sendNotification(transition.isFirstBeat, transition.previousStatus, beat)
			beat.DownCount = 0
		}
	}

	r.persistAndPublishHeartbeat(beat, now)
}

func (r *MonitorRunner) activeMaintenanceWindow(monitor *model.Monitor, now time.Time) (model.MaintenanceWindow, bool) {
	var window model.MaintenanceWindow
	err := r.DB.Where("user_id = ? AND active = ? AND start_at <= ? AND end_at > ?", monitor.UserID, true, now, now).
		Where("monitor_id IS NULL OR monitor_id = ?", monitor.ID).
		Order("monitor_id DESC, start_at DESC").
		First(&window).Error
	return window, err == nil
}

func (r *MonitorRunner) maintenanceBeat(window model.MaintenanceWindow) {
	now := time.Now()
	beat := model.Heartbeat{
		MonitorID: r.Monitor.ID,
		Status:    model.StatusMaintenance,
		Msg:       "maintenance window: " + window.Name,
		Time:      now,
	}
	r.Calculator.Update(beat.Status, nil, now)
	r.persistAndPublishHeartbeat(beat, now)
}

func (r *MonitorRunner) groupBeat() {
	monitor := r.Monitor
	snapshot, err := ComputeMonitorStatus(r.DB, monitor.UserID, monitor.ID)
	if err != nil {
		log.Printf("Group check error for monitor %s: %v", monitor.Name, err)
		return
	}

	now := time.Now()
	beat := model.Heartbeat{
		MonitorID: monitor.ID,
		Status:    snapshot.Status,
		Msg:       GroupStatusMessage(snapshot.Status),
		Time:      now,
	}

	var previousBeat model.Heartbeat
	r.DB.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&previousBeat)

	transition := applyGroupHeartbeatState(&beat, previousBeat)
	if beat.Important {
		r.sendNotification(transition.isFirstBeat, transition.previousStatus, beat)
	}

	r.persistAndPublishHeartbeat(beat, now)
}

func (r *MonitorRunner) persistAndPublishHeartbeat(beat model.Heartbeat, now time.Time) {
	r.Calculator.Update(beat.Status, beat.PingMS, now)
	r.DB.Create(&beat)
	if r.Publisher != nil {
		r.Publisher.SendToUser(r.Monitor.UserID, "heartbeat", beat)
	}
}

func (r *MonitorRunner) shouldResendDownNotification(now time.Time, resendInterval uint32) bool {
	var lastImportant model.Heartbeat
	err := r.DB.Where("monitor_id = ? AND important = ? AND status = ?", r.Monitor.ID, true, model.StatusDown).
		Order("time DESC").
		First(&lastImportant).Error
	if err != nil || lastImportant.ID == 0 {
		return true
	}
	return now.Sub(lastImportant.Time) >= time.Duration(resendInterval)*time.Second
}

func (r *MonitorRunner) sendNotification(isFirstBeat bool, prevStatus uint16, beat model.Heartbeat) {
	dispatch := NewNotifyDispatch(r.DB)
	dispatch.Send(r.Monitor, beat, isFirstBeat, prevStatus)
	dispatch.markIncident(r.DB, r.Monitor.ID, r.Monitor.Name, prevStatus, beat.Status, beat.Msg)
}

func isImportantBeat(isFirstBeat bool, prevStatus, newStatus uint16) bool {
	if isFirstBeat {
		return true
	}
	if prevStatus != newStatus {
		return true
	}
	return false
}
