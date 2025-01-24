package js

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
)

// VM the js runtime.
// An instance of VM can only be used by a single goroutine at a time.
type VM interface {
	// RunModule run the sobek.CyclicModuleRecord.
	// To compile the module, sobek.ParseModule or ModuleLoader.CompileModule
	RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (sobek.Value, error)
	// Run execute the given function in the EventLoop.
	// when context done interrupt VM execution and release the VM.
	// This is usually used when needs to be called repeatedly many times.
	// like this:
	//
	//	func main() {
	//		scheduler := js.NewScheduler(js.SchedulerOptions{
	//			InitialVMs: 2,
	//			Loader:     js.NewModuleLoader(),
	//		})
	//		run := func(ctx context.Context, scheduler js.Scheduler) int64 {
	//			vm, err := scheduler.Get()
	//			if err != nil {
	//				panic(err)
	//			}
	//			rt := vm.Runtime()
	//
	//			module, err := scheduler.Loader().CompileModule("sum", "module.exports = (a, b) => a + b")
	//			if err != nil {
	//				panic(module)
	//			}
	//
	//			sum, err := js.ModuleCallable(rt, module)
	//			if err != nil {
	//				panic(err)
	//			}
	//
	//			var total int64
	//			vm.Run(ctx, func() {
	//				for i := 0; i < 8; i++ {
	//					v, err := sum(sobek.Undefined(), rt.ToValue(i), rt.ToValue(total))
	//					if err != nil {
	//						panic(err)
	//					}
	//					total = v.ToInteger()
	//				}
	//			})
	//
	//			return total
	//		}
	//
	//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	//		defer cancel()
	//
	//		fmt.Println(run(ctx, scheduler))
	//	}
	Run(context.Context, func())
	// Context return the ja context object of NewContext
	Context() sobek.Value
	// Loader return the ModuleLoader
	Loader() ModuleLoader
	// Runtime return the js runtime
	Runtime() *sobek.Runtime
}

type Option func(*vmImpl)

// WithInitial call sobek.Runtime on VM create, be care require and module not working when init.
func WithInitial(fn func(*sobek.Runtime)) Option {
	return func(vm *vmImpl) { fn(vm.runtime) }
}

// WithModuleLoader set a ModuleLoader, if not present require and es module will not work.
func WithModuleLoader(loader ModuleLoader) Option {
	return func(vm *vmImpl) { vm.loader = loader }
}

// NewVM creates a new JavaScript VM
// Initialize the EventLoop, global module, console.
func NewVM(opts ...Option) VM {
	rt := sobek.New()
	rt.SetFieldNameMapper(FieldNameMapper{})
	EnableConsole(rt)
	InitGlobalModule(rt)
	vm := &vmImpl{
		runtime:   rt,
		eventloop: NewEventLoop(),
		ctx:       NewContext(context.Background(), rt),
	}
	for _, opt := range opts {
		opt(vm)
	}
	if vm.release == nil {
		vm.release = func() {}
	}
	if vm.loader == nil {
		vm.loader = new(emptyLoader)
	}

	vm.loader.EnableRequire(rt).EnableImportModuleDynamically(rt)
	_ = rt.GlobalObject().SetSymbol(symbolVM, &vmself{vm})
	_ = rt.GlobalObject().DefineDataProperty("$", vm.ctx, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	return vm
}

type (
	vmImpl struct {
		runtime   *sobek.Runtime
		eventloop *EventLoop
		ctx       sobek.Value
		release   func()
		loader    ModuleLoader
	}

	vmctx struct{ ctx context.Context }

	vmself struct{ vm *vmImpl }
)

// Loader return the ModuleLoader
func (vm *vmImpl) Loader() ModuleLoader { return vm.loader }

// Runtime return the js runtime
func (vm *vmImpl) Runtime() *sobek.Runtime { return vm.runtime }

func (vm *vmImpl) Context() sobek.Value { return vm.ctx }

// RunModule run the sobek.CyclicModuleRecord.
// The module default export must be a function.
func (vm *vmImpl) RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (ret sobek.Value, err error) {
	vm.Run(ctx, func() {
		var call sobek.Callable
		call, err = ModuleCallable(vm.runtime, vm.loader.ResolveModule, module)
		if err != nil {
			return
		}

		values := make([]sobek.Value, len(args))
		for i, arg := range args {
			values[i] = vm.runtime.ToValue(arg)
		}

		ret, err = call(sobek.Undefined(), values...)
	})
	return
}

// Run execute the given function in the EventLoop.
// when context done interrupt VM execution and release the VM.
// This is usually used when needs to be called repeatedly many times.
// like this:
//
//	func main() {
//		scheduler := js.NewScheduler(js.SchedulerOptions{
//			InitialVMs: 2,
//			Loader:     js.NewModuleLoader(),
//		})
//		run := func(ctx context.Context, scheduler js.Scheduler) int64 {
//			vm, err := scheduler.Get()
//			if err != nil {
//				panic(err)
//			}
//			rt := vm.Runtime()
//
//			module, err := scheduler.Loader().CompileModule("sum", "module.exports = (a, b) => a + b")
//			if err != nil {
//				panic(module)
//			}
//
//			sum, err := js.ModuleCallable(rt, module)
//			if err != nil {
//				panic(err)
//			}
//
//			var total int64
//			vm.Run(ctx, func() {
//				for i := 0; i < 8; i++ {
//					v, err := sum(sobek.Undefined(), rt.ToValue(i), rt.ToValue(total))
//					if err != nil {
//						panic(err)
//					}
//					total = v.ToInteger()
//				}
//			})
//
//			return total
//		}
//
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
//		defer cancel()
//
//		fmt.Println(run(ctx, scheduler))
//	}
func (vm *vmImpl) Run(ctx context.Context, task func()) {
	defer func() {
		if x := recover(); x != nil {
			stack := vm.runtime.CaptureCallStack(20, nil)
			buf := new(bytes.Buffer)
			for _, frame := range stack {
				frame.Write(buf)
			}
			ski.Logger(ctx).Error(fmt.Sprintf("vm run error: %s", x),
				slog.String("go_stack", string(debug.Stack())),
				slog.String("js_stack", buf.String()))
		}
		vm.ctx.Export().(*vmctx).ctx = context.Background()
		vm.release()
	}()
	// resets the interrupt flag.
	vm.runtime.ClearInterrupt()
	vm.ctx.Export().(*vmctx).ctx = ctx

	go func() {
		select {
		case <-ctx.Done():
			// interrupt the running JavaScript.
			vm.runtime.Interrupt(ctx.Err())
			// stop the event loop.
			vm.eventloop.Stop()
			return
		}
	}()

	vm.eventloop.Start(task)
}

// NewPromise return a sobek.Promise object.
// The second argument is a long-running asynchronous task that will be executed in a child goroutine.
// The third optional argument is a callback that will be executed in the main goroutine.
// Additional arguments will be ignored.
// like this:
//
//	func main() {
//		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(http.StatusOK)
//			_, _ = w.Write([]byte(`{"foo":"bar"}`))
//		}))
//		defer server.Close()
//
//		vm := js.NewVM()
//		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//		defer cancel()
//
//		fetch := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
//			return rt.ToValue(js.NewPromise(rt,
//				func() (*http.Response, error) { return http.Get(call.Argument(0).String()) },
//				func(res *http.Response, err error) (any, error) {
//					defer res.Body.Close()
//					data, err := io.ReadAll(res.Body)
//					if err != nil {
//						return nil, err
//					}
//					return string(data), nil
//				}))
//		}
//		_ = vm.Runtime().Set("fetch", fetch)
//
//		start := time.Now()
//
//		result, err := vm.Runtime().RunString(ctx, fmt.Sprintf(`fetch("%s")`, server.URL))
//		if err != nil {
//			panic(err)
//		}
//		value, err := js.Unwrap(result)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Println(value)
//		fmt.Println(time.Since(start))
//	}
func NewPromise[T any](runtime *sobek.Runtime, async func() (T, error), then ...func(T, error) (any, error)) *sobek.Promise {
	enqueue := self(runtime).eventloop.EnqueueJob()
	promise, resolve, reject := runtime.NewPromise()
	thenFun := func(r T, e error) (any, error) { return r, e }
	if len(then) > 0 {
		thenFun = then[0]
	}

	go func() {
		defer func() {
			if x := recover(); x != nil {
				reject(x)
			}
		}()
		result, err := async()
		enqueue(func() {
			var value any = result
			value, err = thenFun(result, err)
			if err != nil {
				reject(err)
			} else {
				resolve(value)
			}
		})
	}()

	return promise
}

var (
	reflectTypeCtx    = reflect.TypeOf((*vmctx)(nil))
	reflectTypeVmself = reflect.TypeOf((*vmself)(nil))
	symbolVM          = sobek.NewSymbol("Symbol.__vm__")
)

// NewContext create the js context object
func NewContext(ctx context.Context, rt *sobek.Runtime) *sobek.Object {
	obj := rt.ToValue(&vmctx{ctx}).ToObject(rt)
	proto := rt.NewObject()
	_ = obj.SetPrototype(proto)
	err := FreezeObject(rt, obj)
	if err != nil {
		panic(err)
	}

	_ = proto.Set("get", func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue(toCtx(rt, call.This).Value(call.Argument(0).Export()))
	})
	_ = proto.Set("set", func(call sobek.FunctionCall) sobek.Value {
		e := toCtx(rt, call.This)
		if c, ok := e.(ski.Context); ok {
			c.SetValue(call.Argument(0).Export(), call.Argument(1).Export())
			return rt.ToValue(true)
		}
		return rt.ToValue(false)
	})
	_ = proto.Set("toString", func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue("[context]")
	})
	return obj
}

func toCtx(rt *sobek.Runtime, v sobek.Value) context.Context {
	if v.ExportType() == reflectTypeCtx {
		if u := v.Export().(*vmctx); u != nil && u.ctx != nil {
			return u.ctx
		}
	}
	panic(rt.NewTypeError(`value of "this" must be of type vmctx`))
}

// self get VM self
func self(rt *sobek.Runtime) *vmImpl {
	value := rt.GlobalObject().GetSymbol(symbolVM)
	if value.ExportType() == reflectTypeVmself {
		return value.Export().(*vmself).vm
	}
	panic(rt.NewTypeError(`symbol value of "VM" must be of type vmself, ` +
		`this shouldn't happen, maybe not call from VM.Runtime`))
}
