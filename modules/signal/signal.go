package signal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules"
	"github.com/shiroyk/ski/modules/common"
	"github.com/shiroyk/ski/modules/events"
)

func init() {
	modules.Register("signal", modules.Global{
		"AbortController": new(AbortController),
		"AbortSignal":     new(AbortSignal),
	})
	// AbortSignal inherits EventTarget, the events package does not depend on signal.
	events.SetSignalOptionHandler(func(rt *sobek.Runtime, value sobek.Value, onAbort func()) bool {
		signal, ok := abortSignalOf(value)
		if !ok {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType, `The "signal" argument must be an instance of AbortSignal`))
		}
		return signal.onAbort(onAbort)
	})
}

// AbortController interface represents a controller object
// that allows you to abort one or more Web requests as and when desired.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortController.
type AbortController struct{}

func (a *AbortController) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("abort", a.abort)
	_ = p.DefineAccessorProperty("signal", rt.ToValue(a.signal), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.SetSymbol(sobek.SymToStringTag, rt.ToValue("AbortController"))
	return p
}

func (a *AbortController) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	signal := newAbortSignal(rt, js.Context(rt))
	obj := rt.NewObject()
	_ = obj.SetPrototype(call.This.Prototype())
	_ = obj.SetSymbol(symAbortController, &abortController{signal: signal.ValueOf()})
	return obj
}

func (a *AbortController) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := a.prototype(rt)
	ctor := rt.ToValue(a.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

// signal returns the AbortSignal object associated with this object.
func (*AbortController) signal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toAbortController(rt, call.This).signal)
}

// abort invoking this method will set this object's AbortSignal's aborted flag and signal to
// any observers that the associated activity is to be aborted.
func (*AbortController) abort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := toAbortSignal(rt, toAbortController(rt, call.This).signal)
	// AbortController.abort() runs on the runtime goroutine, so the abort steps
	// run synchronously.
	signal.abort(call.Argument(0))
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
	_ = p.DefineAccessorProperty("onabort", rt.ToValue(a.onabort), rt.ToValue(a.setOnabort), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("throwIfAborted", a.throwIfAborted)
	_ = p.Set("abort", a.abort)
	_ = p.Set("timeout", a.timeout)
	_ = p.SetSymbol(sobek.SymToStringTag, rt.ToValue("AbortSignal"))
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
	// AbortSignal.prototype inherits from EventTarget.prototype.
	_ = proto.SetPrototype(types.Prototype(rt, "EventTarget"))
	return ctor, nil
}

// aborted returns true if this AbortSignal's AbortController has signaled to abort, and false otherwise.
func (*AbortSignal) aborted(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toAbortSignal(rt, call.This).aborted)
}

// reason returns the reason why the signal was aborted.
func (*AbortSignal) reason(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := toAbortSignal(rt, call.This)
	if signal.aborted && signal.reason != nil {
		return signal.reason
	}
	return sobek.Undefined()
}

// onabort returns the event handler registered for the abort event.
func (*AbortSignal) onabort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toAbortSignal(rt, call.This).EventHandler("abort")
}

// setOnabort sets the event handler registered for the abort event.
func (*AbortSignal) setOnabort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := toAbortSignal(rt, call.This)
	var callback sobek.Value
	if value := call.Argument(0); !types.IsNil(value) {
		if _, ok := sobek.AssertFunction(value); !ok {
			panic(rt.NewTypeError("Failed to set the 'onabort' property on 'AbortSignal': The provided value is not of type 'EventListener'."))
		}
		callback = value
	}
	signal.SetEventHandler("abort", callback)
	return sobek.Undefined()
}

// throwIfAborted throws the abort reason when the signal is aborted.
func (*AbortSignal) throwIfAborted(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := toAbortSignal(rt, call.This)
	if !signal.aborted {
		return sobek.Undefined()
	}
	if signal.reason != nil {
		panic(signal.reason)
	}
	panic(rt.NewTypeError("The operation was aborted"))
}

// abort returns an AbortSignal which is already aborted.
func (*AbortSignal) abort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	signal := newAbortSignal(rt, js.Context(rt))
	signal.abort(call.Argument(0))
	return signal.ValueOf()
}

// timeout returns an AbortSignal which will be aborted after the given milliseconds.
func (*AbortSignal) timeout(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	timeout := time.Duration(call.Argument(0).ToInteger()) * time.Millisecond
	ctx, cancel := context.WithTimeout(js.Context(rt), timeout)
	signal := newAbortSignalWith(rt, ctx, func(error) { cancel() })
	if timeout <= 0 {
		// The context is already expired, abort synchronously.
		signal.abort(newTimeoutError(rt))
	}
	return signal.ValueOf()
}

type abortSignal struct {
	events.EventTarget
	rt        *sobek.Runtime
	post      js.Job
	ctx       context.Context
	cancel    context.CancelCauseFunc
	stop      func() bool
	once      sync.Once
	aborted   bool
	reason    sobek.Value
	callbacks []func()
}

func newAbortSignalWith(rt *sobek.Runtime, ctx context.Context, cancel func(error)) *abortSignal {
	proto := types.Prototype(rt, "AbortSignal")
	signal := events.NewEventTargetOf[abortSignal](rt, proto)
	signal.rt = rt
	signal.post = js.PostJob(rt)
	signal.ctx = ctx
	signal.cancel = cancel
	// Any cancellation of the signal context (Go cancel, a canceled parent
	// context or a timeout) updates the JavaScript state on the event loop.
	signal.stop = context.AfterFunc(ctx, signal.abortFromContext)
	return signal
}

// onAbort registers a callback invoked synchronously when the signal aborts.
// It returns false without registering when the signal is already aborted.
func (s *abortSignal) onAbort(callback func()) bool {
	if s.aborted {
		return false
	}
	s.callbacks = append(s.callbacks, callback)
	return true
}

// abortFromContext schedules the abort steps when the signal context ends.
func (s *abortSignal) abortFromContext() {
	s.post(func() error { s.abort(s.reasonFromContext()); return nil })
}

// abort runs the abort steps on the runtime goroutine, so they are visible
// synchronously to the caller.
func (s *abortSignal) abort(reason sobek.Value) {
	s.once.Do(func() {
		if s.stop != nil {
			s.stop()
		}
		if reason == nil || sobek.IsUndefined(reason) {
			reason = newAbortError(s.rt)
		}
		s.aborted = true
		s.reason = reason
		s.cancel(errors.New(reason.String()))

		// Run the abort steps: remove listeners registered with this signal.
		callbacks := s.callbacks
		s.callbacks = nil
		for _, callback := range callbacks {
			callback()
		}

		events.DispatchEvent(s.rt, s, events.NewEvent(s.rt, "abort", events.EventInit{}))
	})
}

// reasonFromContext maps the signal context cause to the abort reason.
func (s *abortSignal) reasonFromContext() sobek.Value {
	cause := context.Cause(s.ctx)
	switch {
	case errors.Is(cause, context.DeadlineExceeded):
		return newTimeoutError(s.rt)
	case cause != nil && !errors.Is(cause, context.Canceled):
		return types.New(s.rt, "Error", s.rt.ToValue(cause.Error()))
	default:
		return newAbortError(s.rt)
	}
}

// newAbortError creates the default abort reason, a DOMException AbortError.
func newAbortError(rt *sobek.Runtime) sobek.Value {
	return common.NewDOMException(rt, "This operation was aborted", "AbortError")
}

// newTimeoutError creates the reason of an aborted timeout signal.
func newTimeoutError(rt *sobek.Runtime) sobek.Value {
	return common.NewDOMException(rt, "The operation was aborted due to timeout", "TimeoutError")
}

var (
	symAbortController = sobek.NewSymbol("Symbol.AbortController")
)

type abortController struct {
	signal *sobek.Object
}

func toAbortController(rt *sobek.Runtime, value sobek.Value) *abortController {
	if o, ok := value.(*sobek.Object); ok {
		if brand := o.GetSymbol(symAbortController); brand != nil {
			if controller, ok := brand.Export().(*abortController); ok {
				return controller
			}
		}
	}
	panic(rt.NewTypeError(`Value of "this" must be of type AbortController`))
}

func toAbortSignal(rt *sobek.Runtime, value sobek.Value) *abortSignal {
	if signal, ok := abortSignalOf(value); ok {
		return signal
	}
	panic(rt.NewTypeError(`Value must be of type AbortSignal`))
}

// IsAbortSignal reports whether the value is an AbortSignal.
func IsAbortSignal(value sobek.Value) bool {
	_, ok := abortSignalOf(value)
	return ok
}

func abortSignalOf(value sobek.Value) (*abortSignal, bool) {
	return events.EventTargetOf[*abortSignal](value)
}

// Context return signal context
func Context(rt *sobek.Runtime, value sobek.Value) context.Context {
	if signal, ok := abortSignalOf(value); ok {
		return signal.ctx
	}
	panic(rt.NewTypeError(`Value must be of type AbortSignal`))
}

// newAbortSignal creates a signal canceled by its own controller.
func newAbortSignal(rt *sobek.Runtime, ctx context.Context) *abortSignal {
	c, cancel := context.WithCancelCause(ctx)
	return newAbortSignalWith(rt, c, cancel)
}

// New creates a new AbortSignal. The returned cancel aborts the signal, it is
// safe to call from any goroutine: the signal context ends immediately and the
// JavaScript state is updated on the event loop.
func New(rt *sobek.Runtime, ctx context.Context) (sobek.Value, context.CancelCauseFunc) {
	signalCtx, cancel := context.WithCancelCause(ctx)
	signal := newAbortSignalWith(rt, signalCtx, cancel)
	return signal.ValueOf(), func(err error) {
		cancel(err)
		// Schedule the JavaScript part right here instead of waiting for the
		// context watcher goroutine, so that a cancellation made from the
		// runtime goroutine is not lost when the event loop is about to stop.
		signal.abortFromContext()
	}
}
