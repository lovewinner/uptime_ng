package engine

import "uptime_ng/internal/model"

type heartbeatTransition struct {
	isFirstBeat    bool
	previousStatus uint16
}

func transitionFromPrevious(previous model.Heartbeat) heartbeatTransition {
	transition := heartbeatTransition{
		isFirstBeat:    previous.ID == 0,
		previousStatus: previous.Status,
	}
	if transition.isFirstBeat {
		transition.previousStatus = model.StatusUP
	}
	return transition
}

func applyCheckHeartbeatState(beat *model.Heartbeat, previous model.Heartbeat, monitor *model.Monitor, result *CheckResult) heartbeatTransition {
	transition := transitionFromPrevious(previous)
	beat.DownCount = previous.DownCount
	beat.Retries = previous.Retries

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

	beat.Important = isImportantBeat(transition.isFirstBeat, transition.previousStatus, beat.Status)
	if beat.Important {
		beat.DownCount = 0
	}
	return transition
}

func applyGroupHeartbeatState(beat *model.Heartbeat, previous model.Heartbeat) heartbeatTransition {
	transition := transitionFromPrevious(previous)
	beat.DownCount = previous.DownCount
	beat.Retries = previous.Retries

	if beat.Status == model.StatusDown {
		beat.DownCount++
	} else {
		beat.DownCount = 0
		beat.Retries = 0
	}

	beat.Important = isImportantBeat(transition.isFirstBeat, transition.previousStatus, beat.Status)
	return transition
}
