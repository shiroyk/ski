package js

import (
	"context"
	"maps"
	"sync"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
)

// Module is what a module needs to return
type Module interface {
	Instantiate(*goja.Runtime) (goja.Value, error)
}

// Global implements the interface will load into global when the VM initialize (InitGlobalModule).
type Global interface {
	Module
	Global() // is it a global module
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	if _, ok := mod.(Global); !ok {
		name = modulePrefix + name
	}
	registry.Lock()
	registry.native[name] = mod
	registry.Unlock()
}

// GetModule get the module
func GetModule(name string) (Module, bool) {
	registry.RLock()
	defer registry.RUnlock()
	module, ok := registry.native[name]
	return module, ok
}

func RemoveModule(name string) {
	registry.Lock()
	delete(registry.native, name)
	registry.Unlock()
}

// AllModule get all module
func AllModule() map[string]Module {
	registry.RLock()
	defer registry.RUnlock()
	return maps.Clone(registry.native)
}

const modulePrefix = "ski/"

var registry = struct {
	sync.RWMutex
	native map[string]Module
}{
	native: make(map[string]Module),
}

type cjsModule struct {
	prg           *goja.Program
	exportedNames []string
	o             sync.Once
}

func (cm *cjsModule) Link() error { return nil }

func (cm *cjsModule) InitializeEnvironment() error { return nil }

func (cm *cjsModule) Instantiate(_ *goja.Runtime) (goja.CyclicModuleInstance, error) {
	return &cjsModuleInstance{m: cm}, nil
}

func (cm *cjsModule) RequestedModules() []string { return nil }

func (cm *cjsModule) Evaluate(_ *goja.Runtime) *goja.Promise { return nil }

func (cm *cjsModule) GetExportedNames(_ ...goja.ModuleRecord) []string { return cm.exportedNames }

func (cm *cjsModule) ResolveExport(exportName string, _ ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      cm,
		BindingName: exportName,
	}, false
}

type cjsModuleInstance struct {
	m       *cjsModule
	exports *goja.Object
}

func (cmi *cjsModuleInstance) HasTLA() bool { return false }

func (cmi *cjsModuleInstance) GetBindingValue(name string) goja.Value {
	if name == "default" {
		if d := cmi.exports.Get("default"); d != nil {
			return d
		}
		return cmi.exports
	}
	return cmi.exports.Get(name)
}

func (cmi *cjsModuleInstance) ExecuteModule(rt *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	f, err := rt.RunProgram(cmi.m.prg)
	if err != nil {
		return nil, err
	}

	jsModule := rt.NewObject()
	cmi.exports = rt.NewObject()
	_ = jsModule.Set("exports", cmi.exports)
	if call, ok := goja.AssertFunction(f); ok {
		jsRequire := rt.Get("require")

		// Run the module source, with "cmi.exports" as "this",
		// "cmi.exports" as the "exports" variable, "jsRequire"
		// as the "require" variable and "jsModule" as the
		// "module" variable (Nodejs capable).
		_, err = call(cmi.exports, cmi.exports, jsRequire, jsModule)
		if err != nil {
			return nil, err
		}
	}

	exports := jsModule.Get("exports")
	if goja.IsNull(exports) {
		return nil, ErrInvalidModule
	}
	cmi.exports = exports.ToObject(rt)
	cmi.m.o.Do(func() {
		cmi.m.exportedNames = cmi.exports.Keys()
	})
	return cmi, nil
}

type goModule struct {
	mod           Module
	once          sync.Once
	exportedNames []string
}

func (gm *goModule) Link() error { return nil }

func (gm *goModule) RequestedModules() []string { return nil }

func (gm *goModule) InitializeEnvironment() error { return nil }

func (gm *goModule) Instantiate(rt *goja.Runtime) (goja.CyclicModuleInstance, error) {
	instance, err := gm.mod.Instantiate(rt)
	if err != nil {
		return nil, err
	}
	exports := instance.ToObject(rt)
	gm.once.Do(func() { gm.exportedNames = exports.Keys() })
	return &goModuleInstance{exports}, nil
}

func (gm *goModule) GetExportedNames(_ ...goja.ModuleRecord) []string {
	return gm.exportedNames
}

func (gm *goModule) ResolveExport(exportName string, _ ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      gm,
		BindingName: exportName,
	}, false
}

func (gm *goModule) Evaluate(_ *goja.Runtime) *goja.Promise { return nil }

type goModuleInstance struct{ *goja.Object }

func (gmi *goModuleInstance) GetBindingValue(name string) goja.Value {
	if gmi.Object == nil {
		return nil
	}
	if name == "default" {
		if v := gmi.Get("default"); v != nil {
			return v
		}
		return gmi.Object
	}
	return gmi.Get(name)
}

func (gmi *goModuleInstance) HasTLA() bool { return false }

func (gmi *goModuleInstance) ExecuteModule(_ *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	return gmi, nil
}

const _js_executor_prefix = "executor/"

type _js_executor map[string]ski.NewExecutor

func (m _js_executor) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	var object *goja.Object
	main, ok := m[""]
	if ok {
		object = rt.ToValue(toJSExec(main)).ToObject(rt)
	} else {
		object = rt.NewObject()
	}
	proto := object.Prototype()
	for k, v := range m {
		if k == "" {
			continue
		}
		_ = proto.Set(k, toJSExec(v))
	}
	return object, nil
}

type _js_exec struct {
	e ski.Executor
}

func (e _js_exec) Exec(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	v, err := e.e.Exec(Context(rt), call.Argument(0).Export())
	if err != nil {
		return goja.Null()
	}
	return rt.ToValue(v)
}

func toJSExec(init ski.NewExecutor) func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
		args := make([]ski.Executor, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			args = append(args, ski.Raw(arg.Export()))
		}
		exec, err := init(args...)
		if err != nil {
			Throw(rt, err)
		}
		return rt.ToValue(_js_exec{exec})
	}
}

// Executor the ski.Executor
type Executor struct{ goja.CyclicModuleRecord }

func new_executor() ski.NewExecutor {
	return ski.StringExecutor(func(str string) (ski.Executor, error) {
		module, err := GetScheduler().Loader().CompileModule("", str)
		if err != nil {
			return nil, err
		}
		return Executor{module}, nil
	})
}

func (p Executor) Exec(ctx context.Context, arg any) (any, error) {
	value, err := RunModule(ski.WithValue(ctx, "content", arg), p)
	if err != nil {
		return nil, err
	}

	unwrap, err := Unwrap(value)
	if err != nil {
		return nil, err
	}
	if s, ok := unwrap.([]any); ok {
		return ski.NewIterator(s), nil
	}
	return unwrap, nil
}
