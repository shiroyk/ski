package plugin

import (
	"github.com/shiroyk/cloudcat/plugin/internal/ext"
)

// GetAll returns all plugins.
func GetAll() []*ext.Extension { return ext.GetAll() }
