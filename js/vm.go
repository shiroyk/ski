package js

import (
	"context"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/shiroyk/cloudcat/ext"
	"github.com/shiroyk/cloudcat/js/modules"
	_ "github.com/shiroyk/cloudcat/js/modules/cache"
	_ "github.com/shiroyk/cloudcat/js/modules/cookie"
	_ "github.com/shiroyk/cloudcat/js/modules/http"
	_ "github.com/shiroyk/cloudcat/js/modules/shortener"
	"github.com/shiroyk/cloudcat/parser"
	"golang.org/x/exp/maps"
)

type Program struct {
	Code string
	Args map[string]any
}

type VM interface {
	Run(context.Context, Program) (goja.Value, error)
	RunString(context.Context, string) (goja.Value, error)
}

type vmImpl struct {
	runtime   *goja.Runtime
	useStrict bool
}

func newVM(useStrict bool) VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	EnableConsole(vm)
	modules.EnableRequire(vm)

	// Enable native modules
	for _, extension := range ext.Get(ext.JSExtension) {
		if native, ok := extension.Module.(modules.NativeModule); ok {
			_ = vm.Set(extension.Name, native.New())
		}
	}

	return &vmImpl{vm, useStrict}
}

func (vm *vmImpl) Run(ctx context.Context, p Program) (goja.Value, error) {
	code := p.Code
	argKeys := maps.Keys(p.Args)
	argValues := make([]goja.Value, 0, len(p.Args))

	for _, v := range maps.Values(p.Args) {
		argValues = append(argValues, vm.runtime.ToValue(v))
	}

	if ctx, ok := ctx.(*parser.Context); ok {
		argKeys = append(argKeys, "cat")
		argValues = append(argValues, vm.runtime.ToValue(NewCat(ctx)))
	}

	code, err := transformCode(code)
	if err != nil {
		return nil, err
	}

	code = `(function(` + strings.Join(argKeys, ", ") + "){\n" + code + "\n})"

	program, err := goja.Compile("", code, vm.useStrict)
	if err != nil {
		return nil, err
	}
	fn, err := vm.runtime.RunProgram(program)
	if err != nil {
		return nil, err
	}

	go func() {
		// Wait for the context to be done
		<-ctx.Done()
		// Interrupt running JavaScript.
		vm.runtime.Interrupt(ctx.Err())
		// Release vm
		GetScheduler().Release(vm)
	}()

	if call, ok := goja.AssertFunction(fn); ok {
		return call(goja.Undefined(), argValues...)
	}

	return nil, ErrVMPoolClosed
}

func (vm *vmImpl) RunString(ctx context.Context, s string) (goja.Value, error) {
	return vm.Run(ctx, Program{Code: s})
}

// transformCode transforms code into return statement
func transformCode(code string) (string, error) {
	jsAst, err := goja.Parse("", code)
	if err != nil {
		return "", err
	}

	statement := jsAst.Body[len(jsAst.Body)-1]
	if _, ok := statement.(*ast.ExpressionStatement); !ok {
		return code, nil
	}

	if len(jsAst.Body) == 1 {
		return "return " + code, nil
	}

	index := statement.Idx0() - 1

	return code[:index] + "return " + code[index:], nil
}
