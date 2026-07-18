package handler

import (
	"testing"
	"time"
)

func TestPeriodRange(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 7, 19, 15, 30, 0, 0, loc) // Sunday

	tests := []struct {
		period    string
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			period:    "day",
			wantStart: time.Date(2026, 7, 19, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 20, 0, 0, 0, 0, loc),
		},
		{
			period:    "week",
			wantStart: time.Date(2026, 7, 13, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 20, 0, 0, 0, 0, loc),
		},
		{
			period:    "month",
			wantStart: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 8, 1, 0, 0, 0, 0, loc),
		},
		{
			period:    "quarter",
			wantStart: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 10, 1, 0, 0, 0, 0, loc),
		},
		{
			period:    "year",
			wantStart: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, loc),
		},
		{
			period:    "unknown",
			wantStart: time.Date(2026, 7, 19, 0, 0, 0, 0, loc),
			wantEnd:   time.Date(2026, 7, 20, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			start, end := periodRange(tt.period, now)
			if !start.Equal(tt.wantStart) || !end.Equal(tt.wantEnd) {
				t.Fatalf("range=%s..%s want %s..%s", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
