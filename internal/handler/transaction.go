package handler

import "gorm.io/gorm"

func runTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
