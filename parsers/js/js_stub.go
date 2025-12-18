//go:build no_jsparser

package js

import (
	"context"
	"errors"
)

func LoadPlugins(ctx context.Context, dir string) error {
	return errors.New("JS parser plugins are not supported in this build")
}

func AddPlugin(ctx context.Context, code string, name string) error {
	return errors.New("JS parser plugins are not supported in this build")
}
