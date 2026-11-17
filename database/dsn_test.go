package database

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func TestSQLiteFileDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "windows drive",
			path: "C:/Program Files/SaveAny/database #1.db",
			want: "file:C:/Program%20Files/SaveAny/database%20%231.db",
		},
		{
			name: "unix absolute",
			path: "/var/lib/saveany/database #1.db",
			want: "file:/var/lib/saveany/database%20%231.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sqliteFileDSN(tt.path); got != tt.want {
				t.Fatalf("sqliteFileDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDSNWithPragmas(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "database #1.db")
	dsn := dsnWithPragmas(path)
	if !strings.HasPrefix(dsn, "file:") {
		t.Fatalf("dsnWithPragmas() = %q, want a file DSN", dsn)
	}
	if strings.HasPrefix(dsn, "file://") {
		t.Fatalf("dsnWithPragmas() = %q, must not use a URI authority", dsn)
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	pragmas := parsed.Query()["_pragma"]
	if len(pragmas) != 3 {
		t.Fatalf("pragmas = %v, want 3 entries", pragmas)
	}
}

func TestDSNWithPragmasOpensDatabase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "database.db")
	db, err := gorm.Open(GetDialect(path), &gorm.Config{Logger: glogger.Discard})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if err := db.Exec("CREATE TABLE probe (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		t.Fatalf("read journal mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal mode = %q, want wal", journalMode)
	}
}
