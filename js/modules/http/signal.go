package http

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// AbortController interface represents a controller object
// that allows you to abort one or more Web requests as and when desired.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortController.
type AbortController struct{}

func (a *AbortController) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("abort", a.abort)
	_ = p.DefineAccessorProperty("signal", rt.ToValue(a.signal), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("AbortController") })
	return p
}

func (a *AbortController) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	signal := new(abortSignal)
	signal.ctx, signal.cancel = context.WithCancelCause(js.Context(rt))
	abortSignalCtor := rt.Get("AbortSignal")
	if abortSignalCtor == nil {
		panic(rt.NewTypeError("AbortSignal is not defined"))
	}
	proto := abortSignalCtor.ToObject(rt).Prototype()
	value := rt.ToValue(signal).ToObject(rt)
	_ = value.SetPrototype(proto)

	obj := rt.ToValue(&abortController{signal: value}).ToObject(rt)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (a *AbortController) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := a.prototype(rt)
	ctor := rt.ToValue(a.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

func (*AbortController) Global() {}

func (*AbortController) signal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toAbortController(rt, call.This).signal)
}

func (*AbortController) abort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	reason := errAbort
	if r := call.Argument(0); !sobek.IsUndefined(r) {
		reason = errors.New(r.String())
	}
	toAbortSignal(rt, toAbortController(rt, call.This).signal).abort(reason)
	return sobek.Undefined()
}

// AbortSignal represents a signal object that allows you to communicate
// with http request and abort it.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
type AbortSignal struct{}

func (a *AbortSignal) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()

	_ = p.DefineAccessorProperty("aborted", rt.ToValue(a.aborted), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("reason", rt.ToValue(a.reason), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("abort", a.abort)
	_ = p.Set("timeout", a.timeout)
	_ = p.SetSymbol(sobek.SymToStringTag, rt.ToValue(func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("AbortSignal") }))
	return p
}

func (a *AbortSignal) constructor(_ sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	panic(rt.NewTypeError("Illegal constructor"))
}

func (a *AbortSignal) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := a.prototype(rt)
	ctor := rt.ToValue(a.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*AbortSignal) Global() {}

func (a *AbortSignal) aborted(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toAbortSignal(rt, call.This)
	return rt.ToValue(this.aborted)
}

func (a *AbortSignal) reason(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toAbortSignal(rt, call.This)
	if this.aborted {
		return rt.ToValue(this.reason)
	}
	return sobek.Undefined()
}

func (a *AbortSignal) abort(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := new(abortSignal)
	signal.ctx, signal.cancel = context.WithCancelCause(context.Background())
	signal.abort(errImmediateAbort)
	object := rt.ToValue(signal).ToObject(rt)
	_ = object.SetPrototype(a.prototype(rt))
	return object
}

func (a *AbortSignal) timeout(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	timeout := call.Argument(0).ToInteger()
	signal := new(abortSignal)
	var cancel context.CancelFunc
	signal.ctx, cancel = context.WithTimeout(js.Context(rt), time.Duration(timeout))
	signal.cancel = func(cause error) { cancel() }
	if timeout <= 0 {
		signal.abort(nil)
	}
	object := rt.ToValue(signal).ToObject(rt)
	_ = object.SetPrototype(a.prototype(rt))
	return object
}

type abortSignal struct {
	ctx     context.Context
	cancel  context.CancelCauseFunc
	once    sync.Once
	aborted bool
	reason  string
}

func (s *abortSignal) String() string { return s.reason }

func (s *abortSignal) abort(reason error) {
	s.once.Do(func() {
		s.aborted = true
		s.cancel(reason)
		if cause := context.Cause(s.ctx); cause != nil {
			s.reason = cause.Error()
		} else if err := s.ctx.Err(); err != nil {
			s.reason = err.Error()
		}
	})
}

type abortController struct {
	signal *sobek.Object
}

var (
	errAbort            = errors.New("aborted")
	errImmediateAbort   = errors.New("immediate abort")
	typeAbortController = reflect.TypeOf((*abortController)(nil))
	typeAbortSignal     = reflect.TypeOf((*abortSignal)(nil))
)

func toAbortController(rt *sobek.Runtime, value sobek.Value) *abortController {
	if value.ExportType() == typeAbortController {
		return value.Export().(*abortController)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type AbortController`))
}

func toAbortSignal(rt *sobek.Runtime, value sobek.Value) *abortSignal {
	if value.ExportType() == typeAbortSignal {
		return value.Export().(*abortSignal)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type AbortSignal`))
}
