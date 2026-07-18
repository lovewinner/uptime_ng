package engine

import "uptime_ng/internal/model"

func applyBucketUpdate(bucket *AggregateBucket, status uint16, pingMS *float64) {
	flatStatus := model.FlatStatus(status)
	if status == model.StatusMaintenance {
		bucket.Maintenance++
		return
	}
	if flatStatus == model.StatusUP {
		bucket.Up++
		if pingMS != nil && *pingMS > 0 {
			updateBucketPingStats(bucket, *pingMS)
		}
		return
	}
	bucket.Down++
}

func updateBucketPingStats(bucket *AggregateBucket, ping float64) {
	if bucket.Up == 1 {
		bucket.AvgPing = ping
		bucket.MinPing = ping
		bucket.MaxPing = ping
		return
	}
	bucket.AvgPing = (bucket.AvgPing*float64(bucket.Up-1) + ping) / float64(bucket.Up)
	if ping < bucket.MinPing {
		bucket.MinPing = ping
	}
	if ping > bucket.MaxPing {
		bucket.MaxPing = ping
	}
}

func bucketFromStats(up uint32, down uint32, avgPing float64, minPing float64, maxPing float64) *AggregateBucket {
	return &AggregateBucket{
		Up:      up,
		Down:    down,
		AvgPing: avgPing,
		MinPing: minPing,
		MaxPing: maxPing,
	}
}

func bucketUptime(bucket *AggregateBucket) float64 {
	return uptimeFromCounts(bucket.Up, bucket.Down)
}

func uptimeFromCounts(up uint32, down uint32) float64 {
	total := up + down
	if total == 0 {
		return 1.0
	}
	return float64(up) / float64(total)
}

func dataPointFromBucket(timestamp int64, bucket *AggregateBucket) DataPoint {
	return DataPoint{
		Timestamp: timestamp,
		Uptime:    bucketUptime(bucket),
		AvgPing:   bucket.AvgPing,
		MinPing:   bucket.MinPing,
		MaxPing:   bucket.MaxPing,
		Up:        bucket.Up,
		Down:      bucket.Down,
	}
}
