package js

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/plugin"
)

var errInitExecutor = errors.New("initializing JavaScript VM executor failed")

// VM the js runtime.
// An instance of VM can only be used by a single goroutine at a time.
type VM interface {
	// RunModule run the goja.CyclicModuleRecord.
	// The module default export must be a function.
	// To compile the module, goja.ParseModule("name", "module", resolver.ResolveModule)
	RunModule(ctx context.Context, module goja.CyclicModuleRecord) (goja.Value, error)
	// RunString run the script string
	RunString(ctx context.Context, src string) (goja.Value, error)
	// Runtime return the js runtime
	Runtime() *goja.Runtime
}

// NewVM creates a new JavaScript VM
// Initialize the EventLoop, require, global module, console.
// If loader.ModuleLoader not declared, use the default loader.NewModuleLoader().
func NewVM() VM {
	rt := goja.New()
	rt.SetFieldNameMapper(FieldNameMapper{})
	InitGlobalModule(rt)
	mr, err := cloudcat.Resolve[ModuleLoader]()
	if err != nil {
		slog.Warn(fmt.Sprintf("ModuleLoader not declared, using default"))
		mr = NewModuleLoader()
		cloudcat.Provide(mr)
	}
	mr.EnableRequire(rt)
	mr.ImportModuleDynamically(rt)
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

	vm := &vmImpl{
		runtime:   rt,
		eventloop: NewEventLoop(rt),
		executor:  executor,
		done:      make(chan struct{}, 1),
		loader:    mr,
	}
	scheduler := cloudcat.ResolveLazy[Scheduler]()
	vm.release = func() {
		s, err := scheduler()
		if err != nil {
			return
		}
		s.Release(vm)
	}
	return vm
}

type (
	vmImpl struct {
		runtime   *goja.Runtime
		eventloop *EventLoop
		executor  goja.Callable
		done      chan struct{}
		release   func()
		loader    ModuleLoader
	}

	vmctx struct{ ctx context.Context }
)

// RunModule run the goja.CyclicModuleRecord.
// The module default export must be a function.
func (vm *vmImpl) RunModule(ctx context.Context, module goja.CyclicModuleRecord) (goja.Value, error) {
	if err := module.Link(); err != nil {
		vm.release()
		return nil, err
	}
	promise := vm.runtime.CyclicModuleRecordEvaluate(module, vm.loader.ResolveModule)
	switch promise.State() {
	case goja.PromiseStateRejected:
		vm.release()
		return nil, promise.Result().Export().(error)
	case goja.PromiseStateFulfilled:
	default:
	}
	value := vm.runtime.GetModuleInstance(module).GetBindingValue("default")
	fn, ok := goja.AssertFunction(value)
	if !ok {
		vm.release()
		return value, nil
	}

	if pc, ok := ctx.(*plugin.Context); ok {
		return vm.run(ctx, fn, NewCtxWrapper(vm, pc))
	}
	return vm.run(ctx, fn)
}

// RunString run the script string
func (vm *vmImpl) RunString(ctx context.Context, src string) (goja.Value, error) {
	if pc, ok := ctx.(*plugin.Context); ok {
		return vm.run(ctx, vm.executor, NewCtxWrapper(vm, pc), vm.runtime.ToValue(src))
	}
	return vm.run(ctx, vm.executor, goja.Undefined(), vm.runtime.ToValue(src))
}

func (vm *vmImpl) run(ctx context.Context, call goja.Callable, args ...goja.Value) (ret goja.Value, err error) {
	// resets the interrupt flag.
	vm.runtime.ClearInterrupt()
	defer func() {
		vm.eventloop.WaitOnRegistered()

		if r := recover(); r != nil {
			stack := vm.runtime.CaptureCallStack(20, nil)
			buf := new(bytes.Buffer)
			for _, frame := range stack {
				frame.Write(buf)
			}
			slog.Error(fmt.Sprintf("vm run error %s", r),
				"stack", string(debug.Stack()), "js stack", buf.String())
		}

		_ = vm.runtime.GlobalObject().Delete("__ctx__")
		vm.done <- struct{}{} // End of run
	}()

	go func() {
		select {
		case <-ctx.Done():
			// Interrupt running JavaScript.
			vm.runtime.Interrupt(ctx.Err())
			// Release vm
			vm.release()
			return
		case <-vm.done:
			// Release vm
			vm.release()
			return
		}
	}()
	_ = vm.runtime.GlobalObject().Set("__ctx__", vmctx{ctx})

	err = vm.eventloop.Start(func() error {
		ret, err = call(goja.Undefined(), args...)
		return err
	})

	return
}

// Runtime return the js runtime
func (vm *vmImpl) Runtime() *goja.Runtime { return vm.runtime }

// NewPromise returns the new promise with the async function.
// must be called on the EventLoop.
// like this:
//
//	func main() {
//		vm := js.NewVM()
//		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//		defer cancel()
//
//		goFunc := func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
//			return rt.ToValue(js.NewPromise(rt, func() (any, error) {
//				time.Sleep(time.Second)
//				return max(call.Argument(0).ToInteger(), call.Argument(1).ToInteger()), nil
//			}))
//		}
//		_ = vm.Runtime().Set("max", goFunc)
//
//		start := time.Now()
//
//		result, err := vm.RunString(ctx, `max(1, 2)`)
//		if err != nil {
//			panic(err)
//		}
//		value, err := js.Unwrap(result)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Println(value)
//		fmt.Println(time.Now().Sub(start))
//	}
func NewPromise[T any](runtime *goja.Runtime, async func() (T, error), then ...func(T, error) (any, error)) *goja.Promise {
	callback := NewEnqueueCallback(runtime)
	promise, resolve, reject := runtime.NewPromise()
	thenFun := func(r T, e error) (any, error) {
		return r, e
	}
	if len(then) > 0 {
		thenFun = then[0]
	}

	go func() {
		result, err := async()
		callback(func() error {
			var value any = result
			value, err = thenFun(result, err)
			if err != nil {
				reject(err)
			} else {
				resolve(value)
			}
			return nil
		})
	}()

	return promise
}

// NewEnqueueCallback signals to the event loop that you are going to do some
// asynchronous work off the main thread and that you may need to execute some
// code back on the main thread when you are done.
// see EventLoop.RegisterCallback.
//
//	func doAsyncWork(runtime *goja.Runtime) *goja.Promise {
//		enqueueCallback := js.NewEnqueueCallback(runtime)
//		promise, resolve, reject := runtime.NewPromise()
//
//		// Do the actual async work in a new independent goroutine, but make
//		// sure that the Promise resolution is done on the main thread:
//
//		go func() {
//			// Also make sure to abort early if the context is cancelled, so
//			// the VM is not stuck when the scenario ends or Ctrl+C is used:
//			result, err := doTheActualAsyncWork()
//			enqueueCallback(func() error {
//				if err != nil {
//					reject(err)
//				} else {
//					resolve(result)
//				}
//				return nil // do not abort the iteration
//			})
//		}()
//		return promise
//	}
func NewEnqueueCallback(runtime *goja.Runtime) EnqueueCallback {
	return runtime.GlobalObject().GetSymbol(enqueueCallbackSymbol).Export().(func() EnqueueCallback)()
}
