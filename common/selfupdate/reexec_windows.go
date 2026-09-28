//go:build windows

package selfupdate

import "errors"

var ErrNoReexec = errors.New("windows cannot replace a running program")

func reexec(_ string, _, _ []string) error {
	return ErrNoReexec
}
