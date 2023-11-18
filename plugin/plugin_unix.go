//go:build unix

package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
)

func LoadPlugin(dir string) (size int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return size, err
	}
	loadErr := make([]error, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		_, err = plugin.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			loadErr = append(loadErr, fmt.Errorf("error opening %s: %v", entry.Name(), err))
			continue
		}
		size++
	}
	return size, errors.Join(loadErr...)
}
