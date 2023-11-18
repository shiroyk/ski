//go:build windows

package plugin

import (
	"log/slog"
)

func LoadPlugin(dir string) (size int, err error) {
	slog.Warn("plugin are only supported on Linux, FreeBSD, and macOS. see https://pkg.go.dev/plugin")
	return
}
