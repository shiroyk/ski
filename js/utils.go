package js

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
)

// Throw js exception
func Throw(rt *sobek.Runtime, err error) {
	var ex *sobek.Exception
	if errors.As(err, &ex) { //nolint:errorlint
		panic(ex)
	}
	panic(rt.ToValue(err))
}

// ToBytes tries to return a byte slice from compatible types.
func ToBytes(data any) ([]byte, error) {
	switch dt := data.(type) {
	case []byte:
		return dt, nil
	case string:
		return []byte(dt), nil
	case sobek.ArrayBuffer:
		return dt.Bytes(), nil
	default:
		return nil, fmt.Errorf("expected string, []byte or ArrayBuffer, but got %T, ", data)
	}
}

// Unwrap the sobek.Value to the raw value
func Unwrap(value sobek.Value) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.Export().(type) {
	default:
		return v, nil
	case sobek.ArrayBuffer:
		return v.Bytes(), nil
	case *sobek.Promise:
		switch v.State() {
		case sobek.PromiseStateRejected:
			return nil, errors.New(v.Result().String())
		case sobek.PromiseStateFulfilled:
			return v.Result().Export(), nil
		default:
			return nil, errors.New("unexpected promise pending state")
		}
	}
}

// ModuleCallable return the sobek.CyclicModuleRecord default export as sobek.Callable.
func ModuleCallable(rt *sobek.Runtime, resolve sobek.HostResolveImportedModuleFunc, module sobek.CyclicModuleRecord) (sobek.Callable, error) {
	instance := rt.GetModuleInstance(module)
	if instance == nil {
		if err := module.Link(); err != nil {
			return nil, err
		}
		promise := rt.CyclicModuleRecordEvaluate(module, resolve)
		switch promise.State() {
		case sobek.PromiseStateRejected:
			return nil, promise.Result().Export().(error)
		case sobek.PromiseStateFulfilled:
		default:
		}
		instance = rt.GetModuleInstance(module)
	}
	value := instance.GetBindingValue("default")
	call, ok := sobek.AssertFunction(value)
	if !ok {
		return nil, errors.New("module default export is not a function")
	}
	return call, nil
}

// Context returns the current context of the sobek.Runtime
func Context(rt *sobek.Runtime) context.Context {
	if v := self(rt).ctx.Export().(*vmctx).ctx; v != nil {
		return v
	}
	return context.Background()
}

// OnDone add a function to execute when the VM has finished running.
// eg: close resources...
func OnDone(rt *sobek.Runtime, job func()) { self(rt).eventloop.OnDone(job) }

// InitGlobalModule init all implement the Global modules
func InitGlobalModule(rt *sobek.Runtime) {
	for name, mod := range AllModule() {
		if mod, ok := mod.(Global); ok {
			instance, err := mod.Instantiate(rt)
			if err != nil {
				slog.Warn(fmt.Sprintf("instantiate global js module %s failed: %s", name, err))
				continue
			}
			_ = rt.Set(name, instance)
		}
	}
}

func FreezeObject(rt *sobek.Runtime, obj sobek.Value) error {
	global := rt.GlobalObject().Get("Object").ToObject(rt)
	freeze, ok := sobek.AssertFunction(global.Get("freeze"))
	if !ok {
		panic("failed to get the Object.freeze function from the runtime")
	}
	_, err := freeze(sobek.Undefined(), obj)
	return err
}

// FieldNameMapper provides custom mapping between Go and JavaScript property names.
type FieldNameMapper struct{}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (FieldNameMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
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
func (FieldNameMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	return strings.ToLower(m.Name[0:1]) + m.Name[1:]
}

// MapValues returns the values of the map m.
// The values will be in an indeterminate order.
func MapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
