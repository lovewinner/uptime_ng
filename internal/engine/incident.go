package engine

import (
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func markIncident(db *gorm.DB, monitorID uint, monitorName string, prevStatus, newStatus uint16, msg string) error {
	if prevStatus == model.StatusUP && (newStatus == model.StatusDown || newStatus == model.StatusPending) {
		incident := model.Incident{
			MonitorID: monitorID,
			Title:     monitorName + " went " + statusLabel(newStatus),
			Status:    model.StatusDown,
			StartedAt: time.Now(),
			Msg:       msg,
		}
		if err := db.Create(&incident).Error; err != nil {
			return err
		}
		log.Printf("Incident created: %s went %s", monitorName, statusLabel(newStatus))
	}

	if (prevStatus == model.StatusDown || prevStatus == model.StatusPending) && newStatus == model.StatusUP {
		var recentIncident model.Incident
		err := db.Where("monitor_id = ? AND status = ?", monitorID, model.StatusDown).
			Order("started_at DESC").First(&recentIncident).Error
		if err == nil {
			now := time.Now()
			recentIncident.EndedAt = &now
			recentIncident.DurationSec = uint32(now.Sub(recentIncident.StartedAt).Seconds())
			recentIncident.Status = model.StatusUP
			recentIncident.Title = monitorName + " recovered"
			if err := db.Save(&recentIncident).Error; err != nil {
				return err
			}
			log.Printf("Incident resolved: %s recovered after %ds", monitorName, recentIncident.DurationSec)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func statusLabel(status uint16) string {
	switch status {
	case model.StatusUP:
		return "UP"
	case model.StatusDown:
		return "DOWN"
	case model.StatusPending:
		return "PENDING"
	default:
		return "UNKNOWN"
	}
}
