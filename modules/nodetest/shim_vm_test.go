package nodetest

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
)

// newVMShim implements vm.runInNewContext() by compiling the code as a function
// with the sandbox keys as parameters, like `new Function(...keys, code)`.
func newVMShim(rt *sobek.Runtime) *sobek.Object {
	runInNewContext := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		code := call.Argument(0).String()
		var args []sobek.Value
		var names []string
		if sandbox := call.Argument(1); !types.IsNil(sandbox) {
			object := sandbox.ToObject(rt)
			for _, key := range object.Keys() {
				if !isIdentifier(key) {
					continue
				}
				names = append(names, key)
				args = append(args, object.Get(key))
			}
		}
		parameters := make([]sobek.Value, 0, len(names)+1)
		for _, name := range names {
			parameters = append(parameters, rt.ToValue(name))
		}
		parameters = append(parameters, rt.ToValue(code))
		function, err := rt.New(rt.Get("Function"), parameters...)
		if err != nil {
			js.Throw(rt, err)
		}
		callable, ok := sobek.AssertFunction(function)
		if !ok {
			panic(rt.NewTypeError("vm.runInNewContext: code is not a function"))
		}
		result, err := callable(sobek.Undefined(), args...)
		if err != nil {
			js.Throw(rt, err)
		}
		return result
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
