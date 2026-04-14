//go:build !sqlite_glebarez

package database

import (
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func GetDialect(dsn string) gorm.Dialector {
	return gormlite.Open(dsn)
}
