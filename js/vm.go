package js

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin"
)

var errInitExecutor = errors.New("initializing JavaScript VM executor failed")

// VM the js runtime.
// An instance of VM can only be used by a single goroutine at a time.
type VM interface {
	// Run the js program
	Run(context.Context, Program) (goja.Value, error)
	// RunString the js string
	RunString(context.Context, string) (goja.Value, error)
	// Runtime the js runtime
	Runtime() *goja.Runtime
}

type vmImpl struct {
	runtime   *goja.Runtime
	eventloop *EventLoop
	executor  goja.Callable
	done      chan struct{}
}

// NewVM creates a new JavaScript VM
// Initialize the EventLoop, require, global module, console
func NewVM(modulePath ...string) VM {
	runtime := goja.New()
	runtime.SetFieldNameMapper(FieldNameMapper{})
	EnableRequire(runtime, modulePath...)
	InitGlobalModule(runtime)
	EnableConsole(runtime)

	// TODO: any better way?
	eval := `(function(ctx, code){with(ctx){return eval(code)}})`
	program := goja.MustCompile("eval", eval, false)
	callable, err := runtime.RunProgram(program)
	if err != nil {
		panic(errInitExecutor)
	}
	executor, ok := goja.AssertFunction(callable)
	if !ok {
		panic(errInitExecutor)
	}

	//keys, _ := runtime.RunString("Object.keys(this)")
	//globalKeys := cast.ToStringSlice(keys.Export())

	return &vmImpl{
		runtime,
		NewEventLoop(runtime),
		executor,
		make(chan struct{}, 1),
	}
}

// Run the js program
func (vm *vmImpl) Run(ctx context.Context, p Program) (ret goja.Value, err error) {
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

		_ = vm.runtime.GlobalObject().DeleteSymbol(vmContextKey)
		vm.done <- struct{}{} // End of run
	}()

	go func() {
		select {
		case <-ctx.Done():
			// Interrupt running JavaScript.
			vm.runtime.Interrupt(ctx.Err())
			// Release vm
			GetScheduler().Release(vm)
			return
		case <-vm.done:
			// Release vm
			GetScheduler().Release(vm)
			return
		}
	}()

	args := p.Args
	if args == nil {
		args = make(map[string]any, 1)
	}
	if ctx, ok := ctx.(*plugin.Context); ok {
		args["cat"] = NewCat(ctx)
	}
	_ = vm.runtime.GlobalObject().SetSymbol(vmContextKey, ctx)

	err = vm.eventloop.Start(func() error {
		ret, err = vm.executor(goja.Undefined(), vm.runtime.ToValue(args), vm.runtime.ToValue(p.Code))
		return err
	})

	return
}

// RunString the js string
func (vm *vmImpl) RunString(ctx context.Context, s string) (goja.Value, error) {
	return vm.Run(ctx, Program{Code: s})
}

// Runtime the js runtime
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
//			return rt.ToValue(NewPromise(rt, func() (any, error) {
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
func NewPromise(runtime *goja.Runtime, asyncFunc func() (any, error)) *goja.Promise {
	callback := NewEnqueueCallback(runtime)
	promise, resolve, reject := runtime.NewPromise()

	go func() {
		result, err := asyncFunc()
		callback(func() error {
			if err != nil {
				reject(err)
			} else {
				resolve(result)
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
