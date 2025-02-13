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

// Global is an interface that marks a Module to be loaded into the global scope when a VM is created.
// Modules implementing this interface will be automatically loaded and their exports will be made available
// as global variables, without requiring explicit imports.
//
// The Loader.InitGlobal method handles the initialization of global modules by:
// 1. Finding all registered modules that implement Global
// 2. Instantiating them using their Instantiate method
// 3. Adding their exports to the global scope of the JavaScript runtime
type Global interface {
	Module
	Global() // mark as global module
}

// Register registers a Module that can be imported in JavaScript code by the given name.
// If the Module implements Global, it will be loaded into the global scope when the VM is created.
// Otherwise, the module will be prefixed with "ski/" and must be explicitly imported.
//
// Example:
//
//	// Register a regular module that must be imported as "ski/mymodule"
//	modules.Register("mymodule", new(MyModule))
//
//	// Register a global module that is automatically loaded
//	modules.Register("console", new(Console))
//
//	type Console struct{}
//	func (Console) Global() {} // Mark as global module
//	func (Console) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
//		// Implementation
//	}
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
