package storagetypes

import (
	"errors"
	"path"
	"strings"
)

var ErrPathEscape = errors.New("storage path escapes the configured base path")

// SecureJoin joins a user-provided storage path to a backend base path.
//
// Storage paths are always relative to the configured base. The helper rejects
// NUL bytes and paths that would climb above the base, and normalizes both
// slash styles so a value that is safe on Windows is also safe on remote
// backends.
func SecureJoin(base, storagePath string) (string, error) {
	if strings.ContainsRune(storagePath, 0) {
		return "", errors.New("storage path contains a NUL byte")
	}

	normalized := strings.ReplaceAll(storagePath, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		cleaned = ""
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrPathEscape
	}

	base = strings.ReplaceAll(base, "\\", "/")
	base = strings.Trim(base, "/")
	base = path.Clean(base)
	if base == "." {
		base = ""
	}
	if base == ".." || strings.HasPrefix(base, "../") {
		return "", ErrPathEscape
	}
	if base == "" {
		return cleaned, nil
	}
	if cleaned == "" {
		return base, nil
	}
	return path.Join(base, cleaned), nil
}

// ErrAlreadyExists is returned by backends that can atomically create a file
// without replacing an existing object.
var ErrAlreadyExists = errors.New("storage path already exists")
