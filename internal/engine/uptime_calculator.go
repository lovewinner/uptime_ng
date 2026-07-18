package engine

import (
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"uptime_ng/internal/model"
)

type UptimeCalculator struct {
	MonitorID uint
	DB        *gorm.DB
	mu        sync.RWMutex

	MinutelyData map[int64]*AggregateBucket
	HourlyData   map[int64]*AggregateBucket
	DailyData    map[int64]*AggregateBucket
}

type AggregateBucket struct {
	Up          uint32
	Down        uint32
	AvgPing     float64
	MinPing     float64
	MaxPing     float64
	Maintenance uint32
}

func NewUptimeCalculator(monitorID uint, db *gorm.DB) *UptimeCalculator {
	return &UptimeCalculator{
		MonitorID:    monitorID,
		DB:           db,
		MinutelyData: make(map[int64]*AggregateBucket),
		HourlyData:   make(map[int64]*AggregateBucket),
		DailyData:    make(map[int64]*AggregateBucket),
	}
}

func (u *UptimeCalculator) Init() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()

	var minutelyBeans []model.StatMinutely
	cutoff := now.Add(-24 * time.Hour)
	if err := u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&minutelyBeans).Error; err != nil {
		return err
	}
	for _, b := range minutelyBeans {
		u.MinutelyData[b.Timestamp] = bucketFromStats(b.Up, b.Down, b.AvgPing, b.MinPing, b.MaxPing)
	}

	var hourlyBeans []model.StatHourly
	cutoff = now.Add(-30 * 24 * time.Hour)
	if err := u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&hourlyBeans).Error; err != nil {
		return err
	}
	for _, b := range hourlyBeans {
		u.HourlyData[b.Timestamp] = bucketFromStats(b.Up, b.Down, b.AvgPing, b.MinPing, b.MaxPing)
	}

	var dailyBeans []model.StatDaily
	cutoff = now.Add(-365 * 24 * time.Hour)
	if err := u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&dailyBeans).Error; err != nil {
		return err
	}
	for _, b := range dailyBeans {
		u.DailyData[b.Timestamp] = bucketFromStats(b.Up, b.Down, b.AvgPing, b.MinPing, b.MaxPing)
	}

	return nil
}

func (u *UptimeCalculator) Update(status uint16, pingMS *float64, date time.Time) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	minutelyKey := u.minutelyKey(date)
	hourlyKey := u.hourlyKey(date)
	dailyKey := u.dailyKey(date)

	minutely := cloneBucket(u.MinutelyData[minutelyKey])
	hourly := cloneBucket(u.HourlyData[hourlyKey])
	daily := cloneBucket(u.DailyData[dailyKey])

	applyBucketUpdate(minutely, status, pingMS)
	applyBucketUpdate(hourly, status, pingMS)
	applyBucketUpdate(daily, status, pingMS)

	if err := u.DB.Transaction(func(tx *gorm.DB) error {
		if err := u.persistMinutelyBucket(tx, minutelyKey, minutely); err != nil {
			return err
		}
		if err := u.persistHourlyBucket(tx, hourlyKey, hourly); err != nil {
			return err
		}
		return u.persistDailyBucket(tx, dailyKey, daily)
	}); err != nil {
		return err
	}

	u.MinutelyData[minutelyKey] = minutely
	u.HourlyData[hourlyKey] = hourly
	u.DailyData[dailyKey] = daily
	return nil
}

func cloneBucket(bucket *AggregateBucket) *AggregateBucket {
	if bucket == nil {
		return &AggregateBucket{}
	}
	cloned := *bucket
	return &cloned
}

func statUpsert() clause.OnConflict {
	return clause.OnConflict{
		Columns:   []clause.Column{{Name: "monitor_id"}, {Name: "timestamp"}},
		DoUpdates: clause.AssignmentColumns([]string{"up", "down", "avg_ping", "min_ping", "max_ping"}),
	}
}

func (u *UptimeCalculator) persistMinutelyBucket(db *gorm.DB, key int64, bucket *AggregateBucket) error {
	return db.Clauses(statUpsert()).Create(&model.StatMinutely{
		MonitorID: u.MonitorID,
		Timestamp: key,
		Up:        bucket.Up,
		Down:      bucket.Down,
		AvgPing:   bucket.AvgPing,
		MinPing:   bucket.MinPing,
		MaxPing:   bucket.MaxPing,
	}).Error
}

func (u *UptimeCalculator) persistHourlyBucket(db *gorm.DB, key int64, bucket *AggregateBucket) error {
	return db.Clauses(statUpsert()).Create(&model.StatHourly{
		MonitorID: u.MonitorID,
		Timestamp: key,
		Up:        bucket.Up,
		Down:      bucket.Down,
		AvgPing:   bucket.AvgPing,
		MinPing:   bucket.MinPing,
		MaxPing:   bucket.MaxPing,
	}).Error
}

func (u *UptimeCalculator) persistDailyBucket(db *gorm.DB, key int64, bucket *AggregateBucket) error {
	return db.Clauses(statUpsert()).Create(&model.StatDaily{
		MonitorID: u.MonitorID,
		Timestamp: key,
		Up:        bucket.Up,
		Down:      bucket.Down,
		AvgPing:   bucket.AvgPing,
		MinPing:   bucket.MinPing,
		MaxPing:   bucket.MaxPing,
	}).Error
}

func (u *UptimeCalculator) minutelyKey(date time.Time) int64 {
	return date.Truncate(time.Minute).Unix()
}

func (u *UptimeCalculator) hourlyKey(date time.Time) int64 {
	return date.Truncate(time.Hour).Unix()
}

func (u *UptimeCalculator) dailyKey(date time.Time) int64 {
	return date.UTC().Truncate(24 * time.Hour).Unix()
}

func (u *UptimeCalculator) Get24HourUptime() float64 {
	return u.getUptimeByType("minute", 1440)
}

func (u *UptimeCalculator) Get30DayUptime() float64 {
	return u.getUptimeByType("day", 30)
}

func (u *UptimeCalculator) Get1YearUptime() float64 {
	return u.getUptimeByType("day", 365)
}

func (u *UptimeCalculator) GetDataPoints(granularity string, num int) []DataPoint {
	u.mu.RLock()
	defer u.mu.RUnlock()

	now := time.Now()
	var points []DataPoint
	step := int64(0)
	var data map[int64]*AggregateBucket

	switch granularity {
	case "minute":
		data = u.MinutelyData
		step = 60
	case "hour":
		data = u.HourlyData
		step = 3600
	default:
		data = u.DailyData
		step = 86400
	}

	endKey := int64(0)
	switch granularity {
	case "minute":
		endKey = u.minutelyKey(now)
	case "hour":
		endKey = u.hourlyKey(now)
	default:
		endKey = u.dailyKey(now)
	}

	startKey := endKey - step*int64(num-1)

	for key := endKey; key >= startKey; key -= step {
		if bucket, ok := data[key]; ok {
			points = append([]DataPoint{dataPointFromBucket(key, bucket)}, points...)
		}
	}

	return points
}

type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Uptime    float64 `json:"uptime"`
	AvgPing   float64 `json:"avg_ping"`
	MinPing   float64 `json:"min_ping"`
	MaxPing   float64 `json:"max_ping"`
	Up        uint32  `json:"up"`
	Down      uint32  `json:"down"`
}

func (u *UptimeCalculator) getUptimeByType(typ string, num int) float64 {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.getUptimeByTypeLocked(typ, num)
}

func (u *UptimeCalculator) getUptimeByTypeLocked(typ string, num int) float64 {
	var totalUP, totalDown uint32

	switch typ {
	case "minute":
		for _, b := range u.MinutelyData {
			totalUP += b.Up
			totalDown += b.Down
		}
	case "hour":
		for _, b := range u.HourlyData {
			totalUP += b.Up
			totalDown += b.Down
		}
	case "day":
		for _, b := range u.DailyData {
			totalUP += b.Up
			totalDown += b.Down
		}
	}

	_ = num
	return uptimeFromCounts(totalUP, totalDown)
}

func (u *UptimeCalculator) CleanupOldData() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()

	minutelyCutoff := now.Add(-24 * time.Hour).Unix()
	if err := u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, minutelyCutoff).
		Delete(&model.StatMinutely{}).Error; err != nil {
		return err
	}
	for k := range u.MinutelyData {
		if k < minutelyCutoff {
			delete(u.MinutelyData, k)
		}
	}

	hourlyCutoff := now.Add(-30 * 24 * time.Hour).Unix()
	if err := u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, hourlyCutoff).
		Delete(&model.StatHourly{}).Error; err != nil {
		return err
	}
	for k := range u.HourlyData {
		if k < hourlyCutoff {
			delete(u.HourlyData, k)
		}
	}

	dailyCutoff := now.Add(-365 * 24 * time.Hour).Unix()
	if err := u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, dailyCutoff).
		Delete(&model.StatDaily{}).Error; err != nil {
		return err
	}
	for k := range u.DailyData {
		if k < dailyCutoff {
			delete(u.DailyData, k)
		}
	}
	return nil
}

type UptimeResult struct {
	Uptime  float64 `json:"uptime"`
	AvgPing float64 `json:"avg_ping"`
}
