package js

import (
	"context"

	"github.com/grafana/sobek"
)

// RunModule the sobek.CyclicModuleRecord
//
// example:
//
//	module, err := js.CompileModule("add", "export default (a, b) => a + b")
//	if err != nil {
//		panic(err)
//	}
//	value, err := js.RunModule(context.Background(), module, 1, 2)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(value.Export()) // 3
func RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (sobek.Value, error) {
	return NewVM().RunModule(ctx, module, args...)
}

// RunString executes the given string
//
// example:
//
//	value, err := js.RunString(context.Background(), `1 + 1`)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(value.Export()) // 2
func RunString(ctx context.Context, str string) (sobek.Value, error) {
	return NewVM().RunString(ctx, str)
}

// RunProgram executes the given sobek.Program
//
// example:
//
//	program, err := sobek.Compile("", `1 + 1`, false)
//	if err != nil {
//		panic(err)
//	}
//	value, err := js.RunProgram(context.Background(), program)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(value.Export()) // 2
func RunProgram(ctx context.Context, program *sobek.Program) (sobek.Value, error) {
	return NewVM().RunProgram(ctx, program)
}

// Run executes the given function
//
// example:
//
//	err := js.Run(context.Background(), func(rt *sobek.Runtime) error {
//		_, err := rt.RunString(`console.log('hello world')`)
//		return err
//	})
//	if err != nil {
//		panic(err)
//	}
func Run(ctx context.Context, fn func(*sobek.Runtime) error) error {
	vm := NewVM()
	return vm.Run(ctx, func() error { return fn(vm.Runtime()) })
}

// CompileModule compile module from source string (cjs/esm).
func CompileModule(name, source string) (sobek.CyclicModuleRecord, error) {
	return Loader().CompileModule(name, source)
}
