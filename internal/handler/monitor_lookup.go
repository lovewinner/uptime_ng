package handler

import (
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func userMonitor(db *gorm.DB, userID uint, monitorID uint) (model.Monitor, error) {
	var monitor model.Monitor
	err := db.Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error
	return monitor, err
}

func userGroupMonitor(db *gorm.DB, userID uint, monitorID uint) (model.Monitor, error) {
	var monitor model.Monitor
	err := db.Where("id = ? AND user_id = ? AND type = ?", monitorID, userID, model.MonitorTypeGroup).First(&monitor).Error
	return monitor, err
}

func userMonitorParentID(db *gorm.DB, userID uint, monitorID uint) (*uint, error) {
	var monitor model.Monitor
	err := db.Select("id", "group_id").Where("id = ? AND user_id = ?", monitorID, userID).First(&monitor).Error
	return monitor.GroupID, err
}

func wouldCreateGroupCycle(db *gorm.DB, userID uint, monitorID uint, parentID uint) (bool, error) {
	seen := map[uint]bool{}
	current := parentID
	for current != 0 {
		if current == monitorID {
			return true, nil
		}
		if seen[current] {
			return true, nil
		}
		seen[current] = true

		parentID, err := userMonitorParentID(db, userID, current)
		if err != nil {
			return false, err
		}
		if parentID == nil {
			return false, nil
		}
		current = *parentID
	}
	return false, nil
}

func userGroupPath(db *gorm.DB, userID uint, groupID *uint) []string {
	if groupID == nil {
		return nil
	}

	path := []string{}
	seen := map[uint]bool{}
	current := *groupID
	for current != 0 {
		if seen[current] {
			break
		}
		seen[current] = true

		group, err := userGroupMonitor(db, userID, current)
		if err != nil {
			break
		}
		path = append([]string{group.Name}, path...)
		if group.GroupID == nil {
			break
		}
		current = *group.GroupID
	}
	return path
}
