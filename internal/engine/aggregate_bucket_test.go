package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestApplyBucketUpdateTracksCountsAndPingStats(t *testing.T) {
	bucket := &AggregateBucket{}
	pingA := 40.0
	pingB := 20.0

	applyBucketUpdate(bucket, model.StatusUP, &pingA)
	applyBucketUpdate(bucket, model.StatusUP, &pingB)
	applyBucketUpdate(bucket, model.StatusDown, nil)
	applyBucketUpdate(bucket, model.StatusMaintenance, nil)

	if bucket.Up != 2 || bucket.Down != 1 || bucket.Maintenance != 1 {
		t.Fatalf("counts up=%d down=%d maintenance=%d", bucket.Up, bucket.Down, bucket.Maintenance)
	}
	if bucket.AvgPing != 30 || bucket.MinPing != 20 || bucket.MaxPing != 40 {
		t.Fatalf("ping avg=%f min=%f max=%f", bucket.AvgPing, bucket.MinPing, bucket.MaxPing)
	}
	if got := bucketUptime(bucket); got != 2.0/3.0 {
		t.Fatalf("uptime=%f", got)
	}
}

func TestDataPointFromBucket(t *testing.T) {
	point := dataPointFromBucket(123, &AggregateBucket{
		Up:      3,
		Down:    1,
		AvgPing: 12,
		MinPing: 5,
		MaxPing: 20,
	})

	if point.Timestamp != 123 || point.Uptime != 0.75 || point.AvgPing != 12 || point.MinPing != 5 || point.MaxPing != 20 {
		t.Fatalf("point=%+v", point)
	}
}
