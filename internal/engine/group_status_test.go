package engine

import (
	"testing"

	"uptime_ng/internal/model"
)

func TestGroupStatusAccumulatorResult(t *testing.T) {
	tests := []struct {
		name       string
		children   []MonitorStatusSnapshot
		pendingErr bool
		wantStatus uint16
		wantUptime float64
	}{
		{name: "empty", wantStatus: model.StatusPending, wantUptime: 1.0},
		{
			name: "all up",
			children: []MonitorStatusSnapshot{
				{Status: model.StatusUP, Uptime24H: 1.0},
				{Status: model.StatusUP, Uptime24H: 0.5},
			},
			wantStatus: model.StatusUP,
			wantUptime: 0.75,
		},
		{
			name: "down wins",
			children: []MonitorStatusSnapshot{
				{Status: model.StatusPending, Uptime24H: 1.0},
				{Status: model.StatusDown, Uptime24H: 0.0},
			},
			wantStatus: model.StatusDown,
			wantUptime: 0.5,
		},
		{
			name: "pending without down",
			children: []MonitorStatusSnapshot{
				{Status: model.StatusUP, Uptime24H: 1.0},
				{Status: model.StatusPending, Uptime24H: 0.8},
			},
			wantStatus: model.StatusPending,
			wantUptime: 0.9,
		},
		{
			name:       "recursive error pending without uptime sample",
			pendingErr: true,
			wantStatus: model.StatusPending,
			wantUptime: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accumulator := groupStatusAccumulator{}
			for _, child := range tt.children {
				accumulator.addChild(child)
			}
			if tt.pendingErr {
				accumulator.addPending()
			}
			status, uptime := accumulator.result()
			if status != tt.wantStatus || uptime != tt.wantUptime {
				t.Fatalf("result=%d,%f want %d,%f", status, uptime, tt.wantStatus, tt.wantUptime)
			}
		})
	}
}
