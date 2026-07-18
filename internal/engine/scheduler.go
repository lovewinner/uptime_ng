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
	if monitor.Type == model.MonitorTypeGroup {
		return
	}

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
	if r.Monitor.Type == model.MonitorTypePush || r.Monitor.Type == model.MonitorTypeGroup {
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

	isFirstBeat := previousBeat.ID == 0
	previousStatus := previousBeat.Status
	if isFirstBeat {
		previousStatus = model.StatusUP
	}

	beat.DownCount = previousBeat.DownCount
	beat.Retries = previousBeat.Retries

	if result.Status == model.StatusDown {
		canRetry := !monitor.RetryOnlyOnStatusCode || result.HTTPStatus > 0
		if canRetry && monitor.MaxRetries > 0 && beat.Retries < monitor.MaxRetries {
			beat.Retries++
			beat.Status = model.StatusPending
		}
		beat.DownCount++
	} else {
		beat.Retries = 0
	}

	isImportant := isImportantBeat(isFirstBeat, previousStatus, beat.Status)
	beat.Important = isImportant

	if isImportant {
		beat.DownCount = 0
		r.sendNotification(isFirstBeat, previousStatus, beat)
	} else if beat.Status == model.StatusDown && monitor.ResendInterval > 0 {
		if r.shouldResendDownNotification(now, monitor.ResendInterval) {
			beat.Important = true
			r.sendNotification(isFirstBeat, previousStatus, beat)
			beat.DownCount = 0
		}
	}

	r.Calculator.Update(beat.Status, beat.PingMS, now)

	r.DB.Create(&beat)

	if r.Publisher != nil {
		r.Publisher.SendToUser(monitor.UserID, "heartbeat", beat)
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
