package js

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vm := NewVM()

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

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := NewVM().RunString(ctx, `while(true){}`)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestVMRunContext(t *testing.T) {
	vm := NewVM()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v, err := vm.RunString(ctx, VMContextKey)
	assert.NoError(t, err)
	assert.Equal(t, ctx, v.Export())
}

func NewTestVM(t *testing.T) VM {
	vm := NewVM()

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
		return vm.ToValue(assert.Equal(t, a, b, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(2).String()))
	})
	_ = vm.Runtime().Set("assert", assertObject)

	return vm
}

func TestNewPromise(t *testing.T) {
	vm := NewTestVM(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	goFunc := func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
		return vm.ToValue(NewPromise(vm, func() (any, error) {
			time.Sleep(time.Second)
			return call.Argument(0).ToInteger() + call.Argument(1).ToInteger(), nil
		}))
	}
	_ = vm.Runtime().Set("asyncAdd", goFunc)

	start := time.Now()

	result, err := vm.RunString(ctx, `asyncAdd(1, 2)`)
	if err != nil {
		assert.NoError(t, err)
	}
	value, err := Unwrap(result)
	if err != nil {
		assert.NoError(t, err)
	}
	assert.EqualValues(t, 3, value)
	assert.EqualValues(t, 1, int(time.Now().Sub(start).Seconds()))
}
