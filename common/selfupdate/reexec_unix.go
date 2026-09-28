//go:build !windows

package selfupdate

import (
	"fmt"
	"syscall"
)

func reexec(exe string, argv, env []string) error {
	if err := syscall.Exec(exe, argv, env); err != nil {
		return fmt.Errorf("failed to exec %s: %w", exe, err)
	}
	return nil
}
