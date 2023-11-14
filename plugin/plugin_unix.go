//go:build unix

package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"plugin"
)

func LoadPlugin(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	loadErr := make([]error, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		_, err = plugin.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			loadErr = append(loadErr, err)
		}
	}
	return errors.Join(loadErr...)
}
