// Package jsmodule the JS module
package jsmodule

import (
	"github.com/shiroyk/cloudcat/plugin/internal/ext"
)

const (
	// ExtPrefix common module prefix
	ExtPrefix = "cloudcat/"
)

// Module is what a module needs to return
type Module interface {
	Exports() any // module instance
}

// Global is it a global module
type Global interface {
	Module
	Global() // is it a global module
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	if _, ok := mod.(any).(Global); !ok {
		name = ExtPrefix + name
	}
	ext.Register(name, ext.JSExtension, mod)
}

func GetModule(name string) (Module, bool) {
	if m, ok := ext.GetName(ext.JSExtension, name); ok {
		return m.Module.(Module), true
	}
	return nil, false
}

func AllModules() map[string]*ext.Extension {
	return ext.Get(ext.JSExtension)
}
