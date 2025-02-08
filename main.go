package ski

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
//	value, err := ski.RunModule(context.Background(), module, 1, 2)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(value.Export()) // 3
func RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (sobek.Value, error) {
	vm, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return vm.RunModule(ctx, module, args...)
}

// RunString executes the given string
//
// example:
//
//	value, err := ski.RunString(context.Background(), `1 + 1`)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(value.Export()) // 2
func RunString(ctx context.Context, str string) (sobek.Value, error) {
	vm, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return vm.RunString(ctx, str)
}
