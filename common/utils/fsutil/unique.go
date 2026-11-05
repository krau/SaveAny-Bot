package fsutil

import (
	"fmt"
	"path"
	"strings"

	"github.com/rs/xid"
)

// UniquePath returns a non-taken path under basePath: the name itself, then
// numbered variants, then a random suffix. The exists callback may return an
// error; callers must not treat a failed check as "not taken".
func UniquePath(basePath, name string, exists func(candidate string) (bool, error), maxAttempts int) (string, error) {
	candidate := path.Join(basePath, name)
	taken, err := exists(candidate)
	if err != nil {
		return "", err
	}
	if !taken {
		return candidate, nil
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= maxAttempts; i++ {
		candidate = path.Join(basePath, fmt.Sprintf("%s_%d%s", stem, i, ext))
		taken, err = exists(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return path.Join(basePath, fmt.Sprintf("%s_%s%s", stem, xid.New().String(), ext)), nil
}
