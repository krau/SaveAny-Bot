package database

import (
	"net/url"
	"path/filepath"
	"strings"
)

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
	// A Windows drive path must be represented as file:///D:/...; without the
	// leading slash net/url emits file://D:/..., which SQLite treats as a UNC
	// path on host "D:".
	slashPath := filepath.ToSlash(absolute)
	if filepath.VolumeName(absolute) != "" && !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	return (&url.URL{Scheme: "file", Path: slashPath, RawQuery: values.Encode()}).String()
}
