package js

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"
	_ "unsafe"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/stretchr/testify/assert"
)

func TestVMContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), "foo", "bar")
	vm := NewVM(WithModuleLoader(NewModuleLoader()))

	v, err := runMod(ctx, vm, `module.exports = (ctx) => ctx.get('foo')`)
	if assert.NoError(t, err) {
		assert.Equal(t, "bar", v.Export())
		assert.Equal(t, context.Background(), Context(vm.Runtime()))
	}
}

func TestVMRunModule(t *testing.T) {
	t.Parallel()
	vm := NewVM()

	testCases := []struct {
		script string
		want   any
	}{
		{"export default () => 1", 1},
		{"export default function () {let a = 1; return a + 1}", 2},
		{"export default async () => 3", 3},
		{"const a = async () => 5; let b = await a(); export default () => b - 1", 4},
		{"export default () => 3 + 2", 5},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			v, err := runMod(context.Background(), vm, c.script)
			if assert.NoError(t, err) {
				vv, err := Unwrap(v)
				if assert.NoError(t, err) {
					assert.EqualValues(t, c.want, vv)
				}
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	start := time.Now()
	_, err := runMod(ctx, NewVM(), "export default () => {while(true){}}")
	took := time.Since(start)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Greater(t, time.Millisecond*300, took)
}

func TestWithInitial(t *testing.T) {
	t.Parallel()
	vm := NewVM(WithInitial(func(rt *goja.Runtime) {
		_ = rt.Set("init", true)
	}))
	v, err := runMod(context.Background(), vm, `export default () => init`)
	if assert.NoError(t, err) {
		assert.Equal(t, true, v.Export())
	}
}

func TestNewPromise(t *testing.T) {
	t.Parallel()
	vm := NewVM()

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

	result, err := runMod(ctx, vm, `export default () => max(1, 2)`)
	if assert.NoError(t, err) {
		value, err := Unwrap(result)
		if assert.NoError(t, err) {
			assert.EqualValues(t, 2, value)
			assert.EqualValues(t, 1, int(time.Now().Sub(start).Seconds()))
		}
	}
}

type testScheduler struct{ vm VM }

func (t *testScheduler) release(vm VM)  { t.vm = vm }
func (*testScheduler) Get() (VM, error) { return nil, nil }
func (*testScheduler) Close() error     { return nil }

func TestVMPanic(t *testing.T) {
	t.Parallel()
	scheduler := new(testScheduler)
	vm := NewVM(func(vm *vmImpl) {
		vm.release = func() { scheduler.release(vm) }
	})

	ctx, cancel := context.WithTimeout(ski.NewContext(context.Background(), nil), time.Second)
	defer cancel()

	log := new(bytes.Buffer)

	logger := slog.New(slog.NewTextHandler(log, nil))

	_ = vm.Runtime().Set("some", func() {
		OnDone(vm.Runtime(), func() { assert.Equal(t, Context(vm.Runtime()), ctx) })
		OnDone(vm.Runtime(), func() { panic("some panic") })
	})
	_, err := runMod(ski.WithLogger(ctx, logger), vm, `export default () => {some(); (() => other.error)()}`)
	if assert.Error(t, err) {
		assert.ErrorContains(t, err, "other is not defined")
		assert.NotNil(t, scheduler.vm)
		assert.Equal(t, context.Background(), Context(vm.Runtime()))
		assert.Contains(t, log.String(), "vm run error: some panic")
	}
}

func runMod(ctx context.Context, vm VM, script string) (goja.Value, error) {
	mod, err := vm.Loader().CompileModule("", script)
	if err != nil {
		return nil, err
	}
	return vm.RunModule(ctx, mod)
}
