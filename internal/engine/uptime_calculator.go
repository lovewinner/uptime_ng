package engine

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"uptime_ng/internal/model"
)

type UptimeCalculator struct {
	MonitorID uint
	DB        *gorm.DB

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
	now := time.Now()

	var minutelyBeans []model.StatMinutely
	cutoff := now.Add(-24 * time.Hour)
	u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&minutelyBeans)
	for _, b := range minutelyBeans {
		u.MinutelyData[b.Timestamp] = &AggregateBucket{
			Up: b.Up, Down: b.Down,
			AvgPing: b.AvgPing, MinPing: b.MinPing, MaxPing: b.MaxPing,
		}
	}

	var hourlyBeans []model.StatHourly
	cutoff = now.Add(-30 * 24 * time.Hour)
	u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&hourlyBeans)
	for _, b := range hourlyBeans {
		u.HourlyData[b.Timestamp] = &AggregateBucket{
			Up: b.Up, Down: b.Down,
			AvgPing: b.AvgPing, MinPing: b.MinPing, MaxPing: b.MaxPing,
		}
	}

	var dailyBeans []model.StatDaily
	cutoff = now.Add(-365 * 24 * time.Hour)
	u.DB.Where("monitor_id = ? AND timestamp > ?", u.MonitorID, cutoff.Unix()).
		Order("timestamp").Find(&dailyBeans)
	for _, b := range dailyBeans {
		u.DailyData[b.Timestamp] = &AggregateBucket{
			Up: b.Up, Down: b.Down,
			AvgPing: b.AvgPing, MinPing: b.MinPing, MaxPing: b.MaxPing,
		}
	}

	return nil
}

func (u *UptimeCalculator) Update(status uint16, pingMS *float64, date time.Time) {
	flatStatus := model.FlatStatus(status)
	ping := 0.0
	if pingMS != nil {
		ping = *pingMS
	}

	minutelyKey := u.minutelyKey(date)
	hourlyKey := u.hourlyKey(date)
	dailyKey := u.dailyKey(date)

	minutely := u.getOrCreate(u.MinutelyData, minutelyKey)
	hourly := u.getOrCreate(u.HourlyData, hourlyKey)
	daily := u.getOrCreate(u.DailyData, dailyKey)

	if status == model.StatusMaintenance {
		minutely.Maintenance++
		hourly.Maintenance++
		daily.Maintenance++
	} else if flatStatus == model.StatusUP {
		minutely.Up++
		hourly.Up++
		daily.Up++

		if ping > 0 {
			u.updatePingStats(minutely, ping)
			u.updatePingStats(hourly, ping)
			u.updatePingStats(daily, ping)
		}
	} else {
		minutely.Down++
		hourly.Down++
		daily.Down++
	}

	u.persistBucket(u.MinutelyData, minutelyKey, minutely, "stat_minutely")
	u.persistBucket(u.HourlyData, hourlyKey, hourly, "stat_hourly")
	u.persistBucket(u.DailyData, dailyKey, daily, "stat_daily")
}

func (u *UptimeCalculator) updatePingStats(bucket *AggregateBucket, ping float64) {
	if bucket.Up == 1 {
		bucket.AvgPing = ping
		bucket.MinPing = ping
		bucket.MaxPing = ping
	} else {
		bucket.AvgPing = (bucket.AvgPing*float64(bucket.Up-1) + ping) / float64(bucket.Up)
		if ping < bucket.MinPing {
			bucket.MinPing = ping
		}
		if ping > bucket.MaxPing {
			bucket.MaxPing = ping
		}
	}
}

func (u *UptimeCalculator) getOrCreate(data map[int64]*AggregateBucket, key int64) *AggregateBucket {
	if bucket, ok := data[key]; ok {
		return bucket
	}
	bucket := &AggregateBucket{}
	data[key] = bucket
	return bucket
}

func (u *UptimeCalculator) persistBucket(data map[int64]*AggregateBucket, key int64, bucket *AggregateBucket, table string) {
	switch table {
	case "stat_minutely":
		u.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "monitor_id"}, {Name: "timestamp"}},
			DoUpdates: clause.AssignmentColumns([]string{"up", "down", "avg_ping", "min_ping", "max_ping"}),
		}).Create(&model.StatMinutely{
			MonitorID: u.MonitorID,
			Timestamp: key,
			Up:        bucket.Up,
			Down:      bucket.Down,
			AvgPing:   bucket.AvgPing,
			MinPing:   bucket.MinPing,
			MaxPing:   bucket.MaxPing,
		})
	case "stat_hourly":
		u.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "monitor_id"}, {Name: "timestamp"}},
			DoUpdates: clause.AssignmentColumns([]string{"up", "down", "avg_ping", "min_ping", "max_ping"}),
		}).Create(&model.StatHourly{
			MonitorID: u.MonitorID,
			Timestamp: key,
			Up:        bucket.Up,
			Down:      bucket.Down,
			AvgPing:   bucket.AvgPing,
			MinPing:   bucket.MinPing,
			MaxPing:   bucket.MaxPing,
		})
	case "stat_daily":
		u.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "monitor_id"}, {Name: "timestamp"}},
			DoUpdates: clause.AssignmentColumns([]string{"up", "down", "avg_ping", "min_ping", "max_ping"}),
		}).Create(&model.StatDaily{
			MonitorID: u.MonitorID,
			Timestamp: key,
			Up:        bucket.Up,
			Down:      bucket.Down,
			AvgPing:   bucket.AvgPing,
			MinPing:   bucket.MinPing,
			MaxPing:   bucket.MaxPing,
		})
	}
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
			total := bucket.Up + bucket.Down
			var uptime float64 = 1.0
			if total > 0 {
				uptime = float64(bucket.Up) / float64(total)
			}
			points = append([]DataPoint{{
				Timestamp: key,
				Uptime:    uptime,
				AvgPing:   bucket.AvgPing,
				Up:        bucket.Up,
				Down:      bucket.Down,
			}}, points...)
		}
	}

	return points
}

type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Uptime    float64 `json:"uptime"`
	AvgPing   float64 `json:"avg_ping"`
	Up        uint32  `json:"up"`
	Down      uint32  `json:"down"`
}

func (u *UptimeCalculator) getUptimeByType(typ string, num int) float64 {
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
	total := totalUP + totalDown
	if total == 0 {
		return 1.0
	}
	return float64(totalUP) / float64(total)
}

func (u *UptimeCalculator) CleanupOldData() {
	now := time.Now()

	minutelyCutoff := now.Add(-24 * time.Hour).Unix()
	u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, minutelyCutoff).
		Delete(&model.StatMinutely{})
	for k := range u.MinutelyData {
		if k < minutelyCutoff {
			delete(u.MinutelyData, k)
		}
	}

	hourlyCutoff := now.Add(-30 * 24 * time.Hour).Unix()
	u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, hourlyCutoff).
		Delete(&model.StatHourly{})
	for k := range u.HourlyData {
		if k < hourlyCutoff {
			delete(u.HourlyData, k)
		}
	}

	dailyCutoff := now.Add(-365 * 24 * time.Hour).Unix()
	u.DB.Where("monitor_id = ? AND timestamp < ?", u.MonitorID, dailyCutoff).
		Delete(&model.StatDaily{})
	for k := range u.DailyData {
		if k < dailyCutoff {
			delete(u.DailyData, k)
		}
	}
}

type UptimeResult struct {
	Uptime  float64 `json:"uptime"`
	AvgPing float64 `json:"avg_ping"`
}
