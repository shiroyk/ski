package plugin

import (
	"os"
	"path/filepath"
	"plugin"

	"github.com/shiroyk/cloudcat/plugin/internal/ext"
)

// GetAll returns all plugins.
func GetAll() []*ext.Extension { return ext.GetAll() }

func LoadPlugin(dir string) []error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []error{err}
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
	return loadErr
}
