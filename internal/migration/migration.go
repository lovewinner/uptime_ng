package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SchemaMigration struct {
	Version   string    `gorm:"primaryKey;size:255" json:"version"`
	Checksum  string    `gorm:"not null;size:64" json:"checksum"`
	AppliedAt time.Time `gorm:"not null" json:"applied_at"`
}

func Apply(db *gorm.DB, dir string) error {
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		checksum := checksum(content)

		var existing SchemaMigration
		err = db.First(&existing, "version = ?", version).Error
		if err == nil {
			if existing.Checksum != checksum {
				return fmt.Errorf("migration %s checksum mismatch", version)
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			statements := splitSQLStatements(string(content))
			for _, statement := range statements {
				if strings.TrimSpace(statement) == "" {
					continue
				}
				if err := tx.Exec(statement).Error; err != nil {
					return fmt.Errorf("%s: %w", version, err)
				}
			}
			return tx.Create(&SchemaMigration{
				Version:   version,
				Checksum:  checksum,
				AppliedAt: time.Now(),
			}).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		next := byte(0)
		if i+1 < len(sql) {
			next = sql[i+1]
		}

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			current.WriteByte(ch)
			if ch == '*' && next == '/' {
				current.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}
		if !inSingleQuote && !inDoubleQuote {
			if ch == '-' && next == '-' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inLineComment = true
				continue
			}
			if ch == '/' && next == '*' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inBlockComment = true
				continue
			}
		}
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}
		if ch == ';' && !inSingleQuote && !inDoubleQuote {
			statements = append(statements, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}
	if strings.TrimSpace(current.String()) != "" {
		statements = append(statements, current.String())
	}
	return statements
}
