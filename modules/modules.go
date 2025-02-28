package modules

import (
	"maps"
	"sync"

	"github.com/grafana/sobek"
)

// Module is the interface that must be implemented by JavaScript modules.
// It defines how a module is instantiated and made available to the JavaScript runtime.
//
// Example implementation:
//
//	func init() {
//		// register a new module named "process"
//		modules.Register("process", new(Process))
//	}
//
//	type Process struct{}
//
//	func (Process) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
//		environ := os.Environ()
//		env := make(map[string]string, len(environ))
//		for _, kv := range environ {
//			k, v, _ := strings.Cut(kv, "=")
//			env[k] = v
//		}
//		ret := rt.NewObject()
//		_ = ret.Set("env", env)
//		return ret, nil
//	}
type Module interface {
	Instantiate(*sobek.Runtime) (sobek.Value, error)
}

// Global represents a collection of related modules grouped under a namespace.
// It maps module names to their Module implementations.
// Global modules are instantiated lazily when first accessed through the global object.
type Global map[string]Module

func (g Global) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.NewObject()
	for name := range g {
		_ = ret.Set(name, rt.Get(name))
	}
	return ret, nil
}

// ModuleFunc type is an adapter to allow the use of ordinary functions as Module.
type ModuleFunc func(sobek.FunctionCall, *sobek.Runtime) sobek.Value

func (m ModuleFunc) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue((func(sobek.FunctionCall, *sobek.Runtime) sobek.Value)(m)), nil
}

// Register registers a Module with the given name and implementation.
// If the module is not a Global module, the name will be prefixed with "ski/".
// The registered modules can later be imported in JavaScript code by name.
//
// Example:
//
//	// Register a regular module that must be imported as "ski/mymodule"
//	modules.Register("mymodule", new(MyModule))
//
//	// Register a global module that can be lazily instantiated
//	modules.Register("sleep", modules.Global{
//		"sleep": modules.ModuleFunc(func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
//			time.Sleep(time.Duration(call.Argument(0).ToInteger()))
//			return sobek.Undefined()
//		}),
//	})
func Register(name string, mod Module) {
	switch mod.(type) {
	case Global:
	default:
		name = prefix + name
	}
	registry.Lock()
	registry.native[name] = mod
	registry.Unlock()
}

// Get the module
func Get(name string) (Module, bool) {
	registry.RLock()
	defer registry.RUnlock()
	module, ok := registry.native[name]
	return module, ok
}

// Remove the modules
func Remove(names ...string) {
	registry.Lock()
	for _, name := range names {
		delete(registry.native, name)
	}
	registry.Unlock()
}

// All get all module
func All() map[string]Module {
	registry.RLock()
	defer registry.RUnlock()
	return maps.Clone(registry.native)
}

const prefix = "ski/"
const nodePrefix = "node:"

var registry = struct {
	sync.RWMutex
	native map[string]Module
}{
	native: make(map[string]Module),
}
