package modules

import (
	"errors"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/ext"
)

const extPrefix = "cloudcat/"

var (
	ErrInvalidModule     = errors.New("invalid module")
	ErrIllegalModuleName = errors.New("illegal module name")

	ErrModuleFileDoesNotExist = errors.New("module file does not exist")
)

// Module is what a module needs to return
type Module interface {
	Exports() any
}

// NativeModule is what a module needs to return
type NativeModule interface {
	New() any
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	ext.Register(extPrefix+name, ext.JSExtension, mod)
}

// RegisterNative the given mod as an external JavaScript module that can be imported
// by name.
func RegisterNative(name string, mod NativeModule) {
	ext.Register(name, ext.JSExtension, mod)
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
