package nodetest

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
)

// newVMShim implements vm.runInNewContext() by running the script with the
// sandbox properties shadowing the globals, node returns the completion value
// of the script, e.g. the value of the last expression.
func newVMShim(rt *sobek.Runtime) *sobek.Object {
	runInNewContext := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		code := call.Argument(0).String()
		global := rt.GlobalObject()
		var restore []func()
		defer func() {
			for i := len(restore) - 1; i >= 0; i-- {
				restore[i]()
			}
		}()
		if sandbox := call.Argument(1); !types.IsNil(sandbox) {
			object := sandbox.ToObject(rt)
			for _, key := range object.Keys() {
				if !isIdentifier(key) {
					continue
				}
				if previous := global.Get(key); previous != nil {
					restore = append(restore, func() { _ = global.Set(key, previous) })
				} else {
					restore = append(restore, func() { _ = global.Delete(key) })
				}
				_ = global.Set(key, object.Get(key))
			}
		}
		value, err := rt.RunString(code)
		if err != nil {
			js.Throw(rt, err)
		}
		return value
	}

	module := rt.NewObject()
	_ = module.Set("runInNewContext", runInNewContext)
	_ = module.Set("runInThisContext", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		value, err := rt.RunString(call.Argument(0).String())
		if err != nil {
			js.Throw(rt, err)
		}
		return value
	})
	_ = module.Set("createContext", func(call sobek.FunctionCall) sobek.Value {
		if sandbox := call.Argument(0); !types.IsNil(sandbox) {
			return sandbox
		}
		return rt.NewObject()
	})
	_ = module.Set("isContext", func(sobek.FunctionCall) sobek.Value { return rt.ToValue(false) })
	return module
}

func isIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_' || r == '$':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
