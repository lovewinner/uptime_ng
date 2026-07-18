package handler

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

func TestRunTransactionCommitsOnSuccess(t *testing.T) {
	db := testDB(t)

	if err := runTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&model.Tag{Name: "prod", Color: "#123456"}).Error
	}); err != nil {
		t.Fatalf("runTransaction: %v", err)
	}

	var count int64
	db.Model(&model.Tag{}).Where("name = ?", "prod").Count(&count)
	if count != 1 {
		t.Fatalf("count=%d want 1", count)
	}
}

func TestRunTransactionRollsBackOnError(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("stop")

	if err := runTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&model.Tag{Name: "prod", Color: "#123456"}).Error; err != nil {
			return err
		}
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}

	var count int64
	db.Model(&model.Tag{}).Where("name = ?", "prod").Count(&count)
	if count != 0 {
		t.Fatalf("count=%d want 0", count)
	}
}

func TestRunTransactionNestedRollback(t *testing.T) {
	db := testDB(t)
	wantErr := errors.New("outer stop")

	err := runTransaction(db, func(tx *gorm.DB) error {
		if err := runTransaction(tx, func(nested *gorm.DB) error {
			return nested.Create(&model.Tag{Name: "prod", Color: "#123456"}).Error
		}); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want %v", err, wantErr)
	}

	var count int64
	db.Model(&model.Tag{}).Where("name = ?", "prod").Count(&count)
	if count != 0 {
		t.Fatalf("count=%d want 0", count)
	}
}
