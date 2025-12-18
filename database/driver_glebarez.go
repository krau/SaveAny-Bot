//go:build sqlite_glebarez

package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func GetDialect(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}
