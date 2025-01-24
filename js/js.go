package js

import (
	"context"

	"github.com/grafana/sobek"
)

// RunModule the sobek.CyclicModuleRecord
//
// example:
//
//	vm, err := js.GetScheduler().Get()
//	if err != nil {
//		panic(err)
//	}
//	module, err := vm.Loader().CompileModule("add", "export default (a, b) => a + b")
//	if err != nil {
//		panic(err)
//	}
//	value, err := vm.RunModule(context.Background(), module, 1, 2)
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
