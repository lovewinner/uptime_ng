package engine

import "uptime_ng/internal/model"

type groupStatusAccumulator struct {
	hasDown    bool
	hasPending bool
	uptimeSum  float64
	uptimeNum  int
}

func (a *groupStatusAccumulator) addChild(child MonitorStatusSnapshot) {
	if child.Status == model.StatusDown {
		a.hasDown = true
	} else if child.Status != model.StatusUP {
		a.hasPending = true
	}
	a.uptimeSum += child.Uptime24H
	a.uptimeNum++
}

func (a *groupStatusAccumulator) addPending() {
	a.hasPending = true
}

func (a groupStatusAccumulator) result() (uint16, float64) {
	if a.uptimeNum == 0 {
		return model.StatusPending, 1.0
	}
	status := model.StatusUP
	if a.hasDown {
		status = model.StatusDown
	} else if a.hasPending {
		status = model.StatusPending
	}
	return status, a.uptimeSum / float64(a.uptimeNum)
}
