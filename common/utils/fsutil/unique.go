package fsutil

import (
	"fmt"
	"path"
	"strings"

	"github.com/rs/xid"
)

// UniquePath returns a path under basePath that is not taken according to
// exists. It tries "name", "name_1.ext", "name_2.ext", ... up to maxAttempts
// numbered candidates, then falls back to "name_<random>.ext". All storage
// backends must use this helper instead of re-implementing unique naming.
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
