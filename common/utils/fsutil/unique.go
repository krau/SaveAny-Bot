package fsutil

import (
	"fmt"
	"path"
	"strings"

	"github.com/rs/xid"
)

// UniquePath returns a non-taken path under basePath: the name itself, then
// numbered variants, then a random suffix.
func UniquePath(basePath, name string, exists func(candidate string) bool, maxAttempts int) string {
	candidate := path.Join(basePath, name)
	if !exists(candidate) {
		return candidate
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= maxAttempts; i++ {
		candidate = path.Join(basePath, fmt.Sprintf("%s_%d%s", stem, i, ext))
		if !exists(candidate) {
			return candidate
		}
	}
	return path.Join(basePath, fmt.Sprintf("%s_%s%s", stem, xid.New().String(), ext))
}
