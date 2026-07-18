package engine

import (
	"errors"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

type HeartbeatPublisher interface {
	SendToUser(userID uint, msgType string, payload any)
}

type Scheduler struct {
	DB              *gorm.DB
	publisher       HeartbeatPublisher
	checkerProvider func(string) Checker
	monitors        map[uint]*MonitorRunner
	mu              sync.RWMutex
	running         bool
}

var errSchedulerStopped = errors.New("scheduler stopped")

type MonitorRunner struct {
	Monitor         *model.Monitor
	Ticker          *time.Ticker
	StopChan        chan struct{}
	DoneChan        chan struct{}
	Calculator      *UptimeCalculator
	DB              *gorm.DB
	Publisher       HeartbeatPublisher
	CheckerProvider func(string) Checker
}

func NewScheduler(db *gorm.DB, publisher HeartbeatPublisher) *Scheduler {
	return &Scheduler{
		DB:              db,
		publisher:       publisher,
		checkerProvider: GetChecker,
		monitors:        make(map[uint]*MonitorRunner),
		running:         true,
	}
}

func (s *Scheduler) StartAll() error {
	var monitors []model.Monitor
	if err := s.DB.Where("active = ?", true).Find(&monitors).Error; err != nil {
		return err
	}

	log.Printf("Scheduler starting %d monitors", len(monitors))

	s.mu.Lock()
	defer s.mu.Unlock()

	prepared := make([]*MonitorRunner, 0, len(monitors))
	for i := range monitors {
		runner, err := s.prepareRunner(&monitors[i])
		if err != nil {
			cleanupPreparedRunners(prepared)
			return err
		}
		if runner != nil {
			prepared = append(prepared, runner)
		}
	}
	for _, runner := range prepared {
		s.startRunnerLocked(runner)
	}

	return nil
}

func (s *Scheduler) StartMonitor(monitor *model.Monitor) error {
	_, err := s.startMonitor(monitor)
	return err
}

func (s *Scheduler) startMonitor(monitor *model.Monitor) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, err := s.prepareRunner(monitor)
	if err != nil || runner == nil {
		return false, err
	}
	s.startRunnerLocked(runner)
	return true, nil
}

func (s *Scheduler) prepareRunner(monitor *model.Monitor) (*MonitorRunner, error) {
	if !s.running {
		return nil, errSchedulerStopped
	}
	if _, exists := s.monitors[monitor.ID]; exists {
		return nil, nil
	}
	if monitor.Type == model.MonitorTypePush {
		return nil, nil
	}

	interval := monitor.Interval
	if interval < model.MinInterval {
		interval = model.DefaultInterval
	}

	calc := NewUptimeCalculator(monitor.ID, s.DB)
	if err := calc.Init(); err != nil {
		return nil, err
	}

	return &MonitorRunner{
		Monitor:         monitor,
		Ticker:          time.NewTicker(time.Duration(interval) * time.Second),
		StopChan:        make(chan struct{}),
		DoneChan:        make(chan struct{}),
		Calculator:      calc,
		DB:              s.DB,
		Publisher:       s.publisher,
		CheckerProvider: s.checkerProvider,
	}, nil
}

func (s *Scheduler) startRunnerLocked(runner *MonitorRunner) {
	s.monitors[runner.Monitor.ID] = runner

	go func() {
		runner.run()
	}()

	log.Printf("Started monitor: %s (id=%d, interval=%ds)", runner.Monitor.Name, runner.Monitor.ID, intervalFromMonitor(runner.Monitor))
}

func cleanupPreparedRunners(runners []*MonitorRunner) {
	for _, runner := range runners {
		runner.Ticker.Stop()
	}
}

func intervalFromMonitor(monitor *model.Monitor) uint32 {
	interval := monitor.Interval
	if interval < model.MinInterval {
		return model.DefaultInterval
	}
	return interval
}

func (s *Scheduler) StopMonitor(monitorID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, exists := s.monitors[monitorID]
	if !exists {
		return
	}

	delete(s.monitors, monitorID)
	stopRunner(runner)
}

func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	for id, runner := range s.monitors {
		delete(s.monitors, id)
		stopRunner(runner)
	}
}

func (s *Scheduler) RestartMonitor(monitor *model.Monitor) error {
	s.StopMonitor(monitor.ID)
	return s.StartMonitor(monitor)
}

func (s *Scheduler) RunningCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.monitors)
}

func (r *MonitorRunner) run() {
	defer close(r.DoneChan)
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

func stopRunner(runner *MonitorRunner) {
	close(runner.StopChan)
	runner.Ticker.Stop()
	<-runner.DoneChan
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
	checkerProvider := r.CheckerProvider
	if checkerProvider == nil {
		checkerProvider = GetChecker
	}
	checker := checkerProvider(monitor.Type)
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
		if err := r.DB.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("dns_last_result", result.Msg).Error; err != nil {
			log.Printf("Failed to update DNS result for monitor %s: %v", monitor.Name, err)
		}
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
	if err := r.DB.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&previousBeat).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query previous heartbeat for monitor %s: %v", monitor.Name, err)
		return
	}

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
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query maintenance window for monitor %s: %v", monitor.Name, err)
	}
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
	if err := r.DB.Where("monitor_id = ?", monitor.ID).Order("time DESC").First(&previousBeat).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query previous group heartbeat for monitor %s: %v", monitor.Name, err)
		return
	}

	transition := applyGroupHeartbeatState(&beat, previousBeat)
	if beat.Important {
		r.sendNotification(transition.isFirstBeat, transition.previousStatus, beat)
	}

	r.persistAndPublishHeartbeat(beat, now)
}

func (r *MonitorRunner) persistAndPublishHeartbeat(beat model.Heartbeat, now time.Time) {
	if err := r.Calculator.Update(beat.Status, beat.PingMS, now); err != nil {
		log.Printf("Failed to update uptime stats for monitor %s: %v", r.Monitor.Name, err)
		return
	}
	if err := r.DB.Create(&beat).Error; err != nil {
		log.Printf("Failed to persist heartbeat for monitor %s: %v", r.Monitor.Name, err)
		return
	}
	if r.Publisher != nil {
		r.Publisher.SendToUser(r.Monitor.UserID, "heartbeat", beat)
	}
}

func (r *MonitorRunner) shouldResendDownNotification(now time.Time, resendInterval uint32) bool {
	var lastImportant model.Heartbeat
	err := r.DB.Where("monitor_id = ? AND important = ? AND status = ?", r.Monitor.ID, true, model.StatusDown).
		Order("time DESC").
		First(&lastImportant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || lastImportant.ID == 0 {
		return true
	}
	if err != nil {
		log.Printf("Failed to query last important heartbeat for monitor %s: %v", r.Monitor.Name, err)
		return false
	}
	return now.Sub(lastImportant.Time) >= time.Duration(resendInterval)*time.Second
}

func (r *MonitorRunner) sendNotification(isFirstBeat bool, prevStatus uint16, beat model.Heartbeat) {
	dispatch := NewNotifyDispatch(r.DB)
	if err := dispatch.Send(r.Monitor, beat, isFirstBeat, prevStatus); err != nil {
		log.Printf("Failed to dispatch notification for monitor %s: %v", r.Monitor.Name, err)
	}
	if err := dispatch.markIncident(r.DB, r.Monitor.ID, r.Monitor.Name, prevStatus, beat.Status, beat.Msg); err != nil {
		log.Printf("Failed to update incident for monitor %s: %v", r.Monitor.Name, err)
	}
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
