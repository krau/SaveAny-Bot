//go:build !sqlite_glebarez

package database

import (
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func GetDialect(dsn string) gorm.Dialector {
	return gormlite.Open(dsn)
}
