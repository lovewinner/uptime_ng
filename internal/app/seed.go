package app

import (
	"log"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

const DefaultAdminUsername = "admin"
const DefaultAdminPassword = "admin123"

func seedDefaultAdmin(db *gorm.DB) {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		log.Printf("Warning: failed to count users for seeding: %v", err)
		return
	}
	if count > 0 {
		return
	}

	hashedPassword, err := model.HashPassword(DefaultAdminPassword)
	if err != nil {
		log.Printf("Warning: failed to hash default admin password: %v", err)
		return
	}

	admin := model.User{
		Username: DefaultAdminUsername,
		Password: hashedPassword,
		Role:     model.RoleAdmin,
		Active:   true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Warning: failed to create default admin user: %v", err)
		return
	}

	log.Printf("Default admin user created: username=%s password=%s (please change on first login)", DefaultAdminUsername, DefaultAdminPassword)
}
