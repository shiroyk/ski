//go:build windows

package plugin

import (
	"golang.org/x/exp/slog"
)

func LoadPlugin(dir string) []error {
	slog.Warn("plugin are only supported on Linux, FreeBSD, and macOS. see https://pkg.go.dev/plugin")
	return nil
}
