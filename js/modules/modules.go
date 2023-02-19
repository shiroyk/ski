package modules

import (
	"errors"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/ext"
)

const extPrefix = "cloudcat/"

var (
	// ErrInvalidModule module is invalid
	ErrInvalidModule = errors.New("invalid module")
	// ErrIllegalModuleName module name is illegal
	ErrIllegalModuleName = errors.New("illegal module name")

	// ErrModuleFileDoesNotExist module not exist
	ErrModuleFileDoesNotExist = errors.New("module file does not exist")
)

// Module is what a module needs to return
type Module interface {
	Exports() any // module instance
	Global() bool // is it a global module
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	if !mod.Global() {
		name = extPrefix + name
	}
	ext.Register(name, ext.JSExtension, mod)
}

// InitGlobalModule init all global modules
func InitGlobalModule(vm *goja.Runtime) {
	// Init global modules
	for _, extension := range ext.Get(ext.JSExtension) {
		if mod, ok := extension.Module.(Module); ok {
			if mod.Global() {
				_ = vm.Set(extension.Name, mod.Exports())
			}
		}
	}
}
