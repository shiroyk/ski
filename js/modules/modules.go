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
	Native() bool // is it a native module
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	if !mod.Native() {
		name = extPrefix + name
	}
	ext.Register(name, ext.JSExtension, mod)
}

// InitNativeModule init all native modules
func InitNativeModule(vm *goja.Runtime) {
	// Init native modules
	for _, extension := range ext.Get(ext.JSExtension) {
		if mod, ok := extension.Module.(Module); ok {
			if mod.Native() {
				_ = vm.Set(extension.Name, mod.Exports())
			}
		}
	}
}

// EnableRequire set runtime require module
func EnableRequire(vm *goja.Runtime, path ...string) {
	rrt := &require{
		vm:            vm,
		nodeModules:   make(map[string]*goja.Object),
		globalFolders: path,
	}

	_ = vm.Set("require", rrt.Require)
}
