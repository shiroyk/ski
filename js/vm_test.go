package js

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/stretchr/testify/assert"
)

func TestVMRunString(t *testing.T) {
	t.Parallel()
	vm := NewTestVM(t)

	testCases := []struct {
		script string
		want   any
	}{
		{"2", 2},
		{"let a = 1; a + 2", 3},
		{"(() => 4)()", 4},
		{"[5]", []any{int64(5)}},
		{"let a = {'key':'foo'}; a", map[string]any{"key": "foo"}},
		{"JSON.stringify({'key':7})", `{"key":7}`},
		{"JSON.stringify([8])", `[8]`},
		{"(async () => 9)()", 9},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			v, err := vm.RunString(context.Background(), c.script)
			assert.NoError(t, err)
			vv, err := Unwrap(v)
			assert.NoError(t, err)
			assert.EqualValues(t, c.want, vv)
		})
	}
}

func TestVMRunModule(t *testing.T) {
	t.Parallel()
	resolver := NewModuleLoader()
	vm := NewTestVM(t, resolver)

	{
		testCases := []struct {
			script string
			want   any
		}{
			{"export default () => 1", 1},
			{"export default function () {let a = 1; return a + 1}", 2},
			{"export default async () => 3", 3},
			{"const a = async () => 5; let b = await a(); export default () => b - 1", 4},
			{"export default 3 + 2", 5},
		}

		for i, c := range testCases {
			module, err := goja.ParseModule(strconv.Itoa(i), c.script, resolver.ResolveModule)
			assert.NoError(t, err)
			t.Run(c.script, func(t *testing.T) {
				v, err := vm.RunModule(context.Background(), module)
				assert.NoError(t, err)
				vv, err := Unwrap(v)
				assert.NoError(t, err)
				assert.EqualValues(t, c.want, vv)
			})
		}
	}
	{
		ctx := plugin.NewContext(plugin.ContextOptions{Values: map[any]any{
			"v1": 1,
			"v2": []string{"2"},
			"v3": map[string]any{"key": 3},
		}})
		testCases := []struct {
			script string
			want   any
		}{
			{"export default (ctx) => ctx.get('v1')", 1},
			{"export default function (ctx) {return ctx.get('v2')[0]}", "2"},
			{"export default async (ctx) => ctx.get('v3').key", 3},
			{"const a = async () => 5; let b = await a(); export default (ctx) => b - ctx.get('v1')", 4},
		}

		for i, c := range testCases {
			module, err := goja.ParseModule(strconv.Itoa(i), c.script, resolver.ResolveModule)
			assert.NoError(t, err)
			t.Run(c.script, func(t *testing.T) {
				v, err := vm.RunModule(ctx, module)
				assert.NoError(t, err)
				vv, err := Unwrap(v)
				assert.NoError(t, err)
				assert.EqualValues(t, c.want, vv)
			})
		}
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	_, err := NewTestVM(t).RunString(ctx, `while(true){}`)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestVMRunWithContext(t *testing.T) {
	t.Parallel()
	{
		vm := NewTestVM(t)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = vm.Runtime().Set("testContext", func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
			return vm.ToValue(VMContext(vm))
		})
		v, err := vm.RunString(ctx, "testContext()")
		assert.NoError(t, err)
		assert.Equal(t, ctx, v.Export())
		assert.Equal(t, context.Background(), VMContext(vm.Runtime()))
	}
	{
		vm := NewTestVM(t)
		ctx := plugin.NewContext(plugin.ContextOptions{})
		_ = vm.Runtime().Set("testContext", func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
			return vm.ToValue(VMContext(vm))
		})
		v, err := vm.RunString(ctx, "testContext()")
		assert.NoError(t, err)
		assert.Equal(t, ctx, v.Export())
		assert.Equal(t, context.Background(), VMContext(vm.Runtime()))
	}
}

func TestNewPromise(t *testing.T) {
	t.Parallel()
	vm := NewTestVM(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	goFunc := func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
		return rt.ToValue(NewPromise(rt, func() (any, error) {
			time.Sleep(time.Second)
			return max(call.Argument(0).ToInteger(), call.Argument(1).ToInteger()), nil
		}))
	}
	_ = vm.Runtime().Set("max", goFunc)

	start := time.Now()

	result, err := vm.RunString(ctx, `max(1, 2)`)
	if err != nil {
		assert.NoError(t, err)
	}
	value, err := Unwrap(result)
	if err != nil {
		assert.NoError(t, err)
	}
	assert.EqualValues(t, 2, value)
	assert.EqualValues(t, 1, int(time.Now().Sub(start).Seconds()))
}

func NewTestVM(t *testing.T, m ...ModuleLoader) VM {
	rt := goja.New()
	rt.SetFieldNameMapper(FieldNameMapper{})
	InitGlobalModule(rt)
	EnableConsole(rt)

	eval := `(ctx, code)=>eval(code)`
	program := goja.MustCompile("<eval>", eval, false)
	callable, err := rt.RunProgram(program)
	if err != nil {
		panic(errInitExecutor)
	}
	executor, ok := goja.AssertFunction(callable)
	if !ok {
		panic(errInitExecutor)
	}
	var ml ModuleLoader
	if len(m) > 0 {
		ml = m[0]
	} else {
		ml = NewModuleLoader()
	}
	ml.EnableRequire(rt)
	ml.ImportModuleDynamically(rt)

	vm := &vmImpl{
		runtime:   rt,
		eventloop: NewEventLoop(rt),
		executor:  executor,
		done:      make(chan struct{}, 1),
		loader:    ml,
		release:   func() {},
	}

	assertObject := vm.Runtime().NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := Unwrap(call.Argument(0))
		if err != nil {
			Throw(vm, err)
		}
		b, err := Unwrap(call.Argument(1))
		if err != nil {
			Throw(vm, err)
		}
		var msg string
		if !goja.IsUndefined(call.Argument(2)) {
			msg = call.Argument(2).String()
		}
		if !assert.Equal(t, b, a, msg) {
			Throw(vm, errors.New("not equal"))
		}
		return
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		var msg string
		if !goja.IsUndefined(call.Argument(1)) {
			msg = call.Argument(1).String()
		}
		if !assert.True(t, call.Argument(0).ToBoolean(), msg) {
			Throw(vm, errors.New("should be true"))
		}
		return
	})
	_ = vm.Runtime().Set("assert", assertObject)

	return vm
}
