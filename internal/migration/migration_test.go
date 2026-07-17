package migration

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyRunsAndTracksMigrations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "001_create_widgets.sql")
	if err := os.WriteFile(file, []byte(`
CREATE TABLE widgets (id integer primary key, name text);
INSERT INTO widgets (name) VALUES ('one;two');
`), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	if err := Apply(db, dir); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if err := Apply(db, dir); err != nil {
		t.Fatalf("apply twice: %v", err)
	}

	var count int64
	if err := db.Table("widgets").Count(&count).Error; err != nil {
		t.Fatalf("query widgets: %v", err)
	}
	if count != 1 {
		t.Fatalf("widgets count=%d want 1", count)
	}
	db.Model(&SchemaMigration{}).Count(&count)
	if count != 1 {
		t.Fatalf("schema migrations count=%d want 1", count)
	}
}

func TestApplyDetectsChecksumMismatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "001_create_widgets.sql")
	if err := os.WriteFile(file, []byte(`CREATE TABLE widgets (id integer primary key);`), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	if err := Apply(db, dir); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if err := os.WriteFile(file, []byte(`CREATE TABLE widgets (id integer primary key, name text);`), 0o644); err != nil {
		t.Fatalf("rewrite migration: %v", err)
	}
	if err := Apply(db, dir); err == nil {
		t.Fatalf("expected checksum mismatch")
	}
}
