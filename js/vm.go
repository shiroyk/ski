package js

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
)

// VM the js runtime.
// An instance of VM can only be used by a single goroutine at a time.
type VM interface {
	// RunModule run the sobek.CyclicModuleRecord.
	// To compile the module, sobek.ParseModule or CompileModule.
	// Any additional arguments are passed to the default export function arguments.
	RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (sobek.Value, error)
	// RunString executes the given string
	RunString(ctx context.Context, str string) (sobek.Value, error)
	// Run execute the given function in the EventLoop.
	// when context done interrupt VM execution and release the VM.
	// This is usually used when needs to be called repeatedly many times.
	// like this:
	//
	//	func main() {
	//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	//		defer cancel()
	//
	//		vm := js.NewVM()
	//		rt := vm.Runtime()
	//
	//		module, err := js.CompileModule("add", "export default (a, b) => a + b")
	//		if err != nil {
	//			panic(module)
	//		}
	//
	//		add, err := js.ModuleCallable(rt, module)
	//		if err != nil {
	//			panic(err)
	//		}
	//
	//		var total int64
	//		vm.Run(ctx, func() error {
	//			for i := 0; i < 8; i++ {
	//				v, err := add(sobek.Undefined(), rt.ToValue(i), rt.ToValue(total))
	//				if err != nil {
	//					panic(err)
	//				}
	//				total = v.ToInteger()
	//			}
	//			return nil
	//		})
	//
	//		fmt.Println(total)
	//	}
	Run(context.Context, func() error) error
	// Runtime return the js runtime
	Runtime() *sobek.Runtime
}

type Option func(*vmImpl)

// WithInitial call on VM create.
func WithInitial(fn func(*sobek.Runtime)) Option {
	return func(vm *vmImpl) { fn(vm.runtime) }
}

// WithRelease call on VM run finish.
func WithRelease(fn func(VM)) Option {
	return func(vm *vmImpl) {
		if prev := vm.release; prev != nil {
			vm.release = func() { prev(); fn(vm) }
		} else {
			vm.release = func() { fn(vm) }
		}
	}
}

// NewVM creates a new JavaScript VM
// Initialize the EventLoop, global module, console.
func NewVM(opts ...Option) VM {
	rt := sobek.New()
	rt.SetFieldNameMapper(fieldNameMapper{})
	EnableConsole(rt)
	Loader().EnableRequire(rt).EnableImportModuleDynamically(rt)

	// init global modules
	for name, mod := range modules.All() {
		if mod, ok := mod.(modules.Global); ok {
			instance, err := mod.Instantiate(rt)
			if err != nil {
				slog.Warn(fmt.Sprintf("instantiate global js module %s failed: %s", name, err))
				continue
			}
			if instance == nil {
				continue
			}
			_ = rt.Set(name, instance)
		}
	}

	vm := &vmImpl{
		runtime:   rt,
		eventloop: NewEventLoop(),
		vmctx:     &vmctx{context.Background()},
	}
	for _, opt := range opts {
		opt(vm)
	}
	if vm.release == nil {
		vm.release = func() {}
	}

	_ = rt.GlobalObject().SetSymbol(symbolVM, &vmself{vm})
	_ = rt.GlobalObject().DefineDataProperty("$ctx", jsContext(vm.vmctx, rt),
		sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	return vm
}

type (
	vmImpl struct {
		*vmctx
		runtime   *sobek.Runtime
		eventloop *EventLoop
		release   func()
	}

	vmctx struct{ ctx context.Context }

	vmself struct{ vm *vmImpl }
)

// Runtime return the js runtime
func (vm *vmImpl) Runtime() *sobek.Runtime { return vm.runtime }

// RunModule run the sobek.CyclicModuleRecord.
// To compile the module, sobek.ParseModule or CompileModule.
// Any additional arguments are passed to the default export function arguments.
func (vm *vmImpl) RunModule(ctx context.Context, module sobek.CyclicModuleRecord, args ...any) (ret sobek.Value, err error) {
	err = vm.Run(ctx, func() error {
		instance, err := ModuleInstance(vm.runtime, module)
		if err != nil {
			return err
		}

		call, ok := sobek.AssertFunction(instance.GetBindingValue("default"))
		if !ok {
			ret = sobek.Undefined()
			return nil
		}

		values := make([]sobek.Value, len(args))
		for i, arg := range args {
			values[i] = vm.runtime.ToValue(arg)
		}

		ret, err = call(sobek.Undefined(), values...)
		return err
	})
	return
}

// RunString executes the given string
func (vm *vmImpl) RunString(ctx context.Context, str string) (ret sobek.Value, err error) {
	err = vm.Run(ctx, func() error {
		ret, err = vm.runtime.RunString(str)
		return err
	})
	return
}

// Run execute the given function in the EventLoop.
// when context done interrupt VM execution and release the VM.
// This is usually used when needs to be called repeatedly many times.
// like this:
//
//	func main() {
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
//		defer cancel()
//
//		vm := js.NewVM()
//		rt := vm.Runtime()
//
//		module, err := js.CompileModule("add", "export default (a, b) => a + b")
//		if err != nil {
//			panic(module)
//		}
//
//		add, err := js.ModuleCallable(rt, module)
//		if err != nil {
//			panic(err)
//		}
//
//		var total int64
//		vm.Run(ctx, func() error {
//			for i := 0; i < 8; i++ {
//				v, err := add(sobek.Undefined(), rt.ToValue(i), rt.ToValue(total))
//				if err != nil {
//					panic(err)
//				}
//				total = v.ToInteger()
//			}
//			return nil
//		})
//
//		fmt.Println(total)
//	}
func (vm *vmImpl) Run(ctx context.Context, task func() error) (err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
			} else {
				err = fmt.Errorf(`%s`, x)
			}
			Logger(ctx).Error(err.Error(), slog.String("stack", string(debug.Stack())))
		}
		vm.ctx = context.Background()
		vm.release()
	}()
	// resets the interrupt flag.
	vm.runtime.ClearInterrupt()
	vm.ctx = ctx

	if _, ok := ctx.Deadline(); ok {
		go func() {
			select {
			case <-ctx.Done():
				// interrupt the running JavaScript.
				vm.runtime.Interrupt(ctx.Err())
				// stop the event loop.
				vm.eventloop.Stop()
			}
		}()
	}

	return vm.eventloop.Start(task)
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
//					if err != nil {
//						return nil, err
//					}
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
//		result, err := vm.RunString(ctx, fmt.Sprintf(`fetch("%s")`, server.URL))
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
func NewPromise[T any](rt *sobek.Runtime, async func() (T, error), then ...func(T, error) (any, error)) *sobek.Promise {
	enqueue := self(rt).eventloop.EnqueueJob()
	promise, resolve, reject := rt.NewPromise()

	thenFun := func(r T, e error) (any, error) { return r, e }
	if len(then) > 0 {
		thenFun = then[0]
	}

	go func() {
		defer func() {
			if x := recover(); x != nil {
				err := reject(x)
				if err != nil {
					slog.Warn(fmt.Sprintf(`reject failed: %s`, err))
				}
			}
		}()

		result, err := async()
		enqueue(func() error {
			value, err := thenFun(result, err)
			if err != nil {
				return reject(err)
			} else {
				return resolve(value)
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

func jsContext(ctx *vmctx, rt *sobek.Runtime) *sobek.Object {
	obj := rt.ToValue(ctx).ToObject(rt)
	proto := rt.NewObject()
	_ = obj.SetPrototype(proto)
	err := FreezeObject(rt, obj)
	if err != nil {
		panic(err)
	}

	_ = proto.Set("toString", func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue("[context]")
	})

	proxy := rt.NewProxy(obj, &sobek.ProxyTrapConfig{
		Get: func(target *sobek.Object, property string, receiver sobek.Value) (value sobek.Value) {
			return rt.ToValue(toCtx(rt, target).Value(property))
		},
		Set: func(target *sobek.Object, property string, value sobek.Value, receiver sobek.Value) (success bool) {
			ctx2 := toCtx(rt, target)
			if c, ok := ctx2.(interface{ SetValue(key, value any) }); ok {
				c.SetValue(property, value.Export())
				return true
			}
			return
		},
	})
	return rt.ToValue(proxy).ToObject(rt)
}

func toCtx(rt *sobek.Runtime, v sobek.Value) context.Context {
	if v.ExportType() == reflectTypeCtx {
		if u := v.Export().(*vmctx); u != nil && u.ctx != nil {
			return u.ctx
		}
	}
	panic(rt.NewTypeError("value of this must be of type vmctx"))
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

// fieldNameMapper provides custom mapping between Go and JavaScript property names.
type fieldNameMapper struct{}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (fieldNameMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
	if v, ok := f.Tag.Lookup("js"); ok {
		if v == "-" {
			return ""
		}
		return v
	}
	return strings.ToLower(f.Name[0:1]) + f.Name[1:]
}

// MethodName returns a JavaScript name for the given method in the given type.
// If this method returns "" the method becomes hidden.
func (fieldNameMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	return strings.ToLower(m.Name[0:1]) + m.Name[1:]
}
