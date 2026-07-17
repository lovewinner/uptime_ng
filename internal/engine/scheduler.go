package engine

import (
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/handler"
	"uptime_ng/internal/model"
)

type Scheduler struct {
	DB       *gorm.DB
	monitors map[uint]*MonitorRunner
	mu       sync.RWMutex
	running  bool
}

type MonitorRunner struct {
	Monitor    *model.Monitor
	Ticker     *time.Ticker
	StopChan   chan struct{}
	Calculator *UptimeCalculator
	DB         *gorm.DB
}

func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		DB:       db,
		monitors: make(map[uint]*MonitorRunner),
		running:  true,
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

	beat.DownCount = previousBeat.DownCount

	if result.Status == model.StatusDown {
		if monitor.MaxRetries > 0 && beat.Retries < monitor.MaxRetries {
			beat.Retries++
			beat.Status = model.StatusPending
		}
		beat.DownCount++
	}

	if monitor.UpsideDown && result.Status == model.StatusUP {
		beat.DownCount++
	} else {
		beat.Retries = 0
	}

	isImportant := isImportantBeat(isFirstBeat, previousStatus, beat.Status)
	beat.Important = isImportant

	if isImportant {
		beat.DownCount = 0
		r.sendNotification(isFirstBeat, beat)
	} else if beat.Status == model.StatusDown && monitor.ResendInterval > 0 {
		if beat.DownCount >= monitor.ResendInterval {
			r.sendNotification(isFirstBeat, beat)
			beat.DownCount = 0
		}
	}

	r.Calculator.Update(beat.Status, beat.PingMS, now)

	r.DB.Create(&beat)

	if handler.Hub != nil {
		handler.Hub.SendToUser(monitor.UserID, "heartbeat", beat)
	}
}

func (r *MonitorRunner) sendNotification(isFirstBeat bool, beat model.Heartbeat) {
	dispatch := NewNotifyDispatch(r.DB)
	var prevStatus uint16 = model.StatusUP
	if !isFirstBeat {
		var pb model.Heartbeat
		r.DB.Where("monitor_id = ?", r.Monitor.ID).Order("time DESC").Offset(1).First(&pb)
		if pb.ID > 0 {
			prevStatus = pb.Status
		}
	}
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