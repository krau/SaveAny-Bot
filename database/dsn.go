package database

import (
	"net/url"
	"path/filepath"
	"strings"
)

func sqliteFileDSN(path string) string {
	return "file:" + (&url.URL{Path: path}).EscapedPath()
}

func dsnWithPragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	absolute, err := filepath.Abs(dsn)
	if err != nil {
		return dsn
	}
	values := url.Values{}
	values.Add("_pragma", "foreign_keys(1)")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "busy_timeout(5000)")
	// SQLite keeps a leading slash from file:///D:/... and treats file://D:/...
	// as a UNC path. Prepending file: directly preserves the native D:/... path.
	return sqliteFileDSN(filepath.ToSlash(absolute)) + "?" + values.Encode()
}
