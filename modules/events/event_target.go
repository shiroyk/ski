package events

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync/atomic"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/common"
)

// EventTarget defines the DOM EventTarget interface.
// The exported methods are the event source facing API, the unexported ones are
// the internal contract: they keep the interface sealed, a type outside this
// package can only satisfy it by embedding a real EventTarget instance.
// https://dom.spec.whatwg.org/#interface-eventtarget
type EventTarget interface {
	// ValueOf returns the JavaScript object of the target.
	ValueOf() *sobek.Object
	// EventHandler returns the event handler attribute for the given type, e.g. onabort.
	EventHandler(typ string) sobek.Value
	// SetEventHandler sets the event handler attribute for the given type, e.g. onabort.
	SetEventHandler(typ string, callback sobek.Value)

	invoke(*event)
	contains(string, *listener) bool
	add(string, *listener)
	remove(string, *listener)
	removeMatching(string, sobek.Value, bool)
}

// NewEventTarget creates a new EventTarget
// prototype.
func NewEventTarget(rt *sobek.Runtime) EventTarget {
	return newEventTarget(rt, nil)
}

// NewEventTargetOf creates a new EventTarget of type E, proto nil means use the
// EventTarget prototype. E must embed an EventTarget field, it is set to the new
// target so a Go defined subclass can hold its own state, and the JavaScript
// object is branded with *E, so EventTargetOf[*E](value) returns it.
func NewEventTargetOf[E EventTarget](rt *sobek.Runtime, proto *sobek.Object) *E {
	var e E
	brand, ok := any(&e).(EventTarget)
	if !ok {
		panic(fmt.Sprintf("events: %T must implement EventTarget", &e))
	}
	target := newEventTarget(rt, proto)
	// Brand the JavaScript object with the most derived target.
	_ = target.ValueOf().SetSymbol(symEventTarget, brand)
	holder := reflect.ValueOf(&e).Elem().FieldByName("EventTarget")
	if !holder.IsValid() || !holder.CanSet() {
		panic(fmt.Sprintf("events: %T must embed an events.EventTarget field", e))
	}
	holder.Set(reflect.ValueOf(EventTarget(target)))
	return &e
}

// DispatchEvent dispatches the event to the target and returns false when the
// event is canceled.
// https://dom.spec.whatwg.org/#concept-event-dispatch
func DispatchEvent(rt *sobek.Runtime, target EventTarget, event Event) bool {
	evt := event.base()
	if evt.dispatching {
		panic(common.NewErrorf(rt, ErrEventRecursion, "The event is already being dispatched"))
	}

	evt.dispatching = true
	evt.target = target
	evt.currentTarget = target
	evt.phase = eventPhaseAtTarget
	evt.propagationStopped = false
	evt.immediatePropagationStopped = false
	evt.inPassiveListener = false
	evt.path = []EventTarget{target}

	defer func() {
		evt.dispatching = false
		evt.path = nil
		evt.currentTarget = nil
		evt.phase = eventPhaseNone
		evt.propagationStopped = false
		evt.immediatePropagationStopped = false
		evt.inPassiveListener = false
	}()

	target.invoke(evt)

	return !evt.defaultPrevented
}

// SignalOptionHandler handles the `signal` option of addEventListener.
// It is installed by the signal module through SetSignalOptionHandler.
//
// It must panic with a TypeError when signal is not an AbortSignal. onAbort is
// invoked synchronously when the signal aborts, and events uses it to remove the
// listener. Returning false means the signal was already aborted, in which case
// the listener must not be registered at all.
type SignalOptionHandler func(rt *sobek.Runtime, signal sobek.Value, onAbort func()) bool

var signalOptionHandler SignalOptionHandler

// SetSignalOptionHandler installs the handler for the `signal` option of addEventListener.
func SetSignalOptionHandler(handler SignalOptionHandler) {
	signalOptionHandler = handler
}

const ErrEventRecursion common.ErrCode = "ERR_EVENT_RECURSION"

// listener is an _event listener record.
// https://dom.spec.whatwg.org/#concept-event-listener
type listener struct {
	callback sobek.Value
	capture  bool
	once     bool
	passive  bool
	removed  atomic.Bool
}

func (l *listener) matches(callback sobek.Value, capture bool) bool {
	return l.capture == capture && l.callback.StrictEquals(callback)
}

// handler is an _event handler attribute, e.g. onabort.
type handler struct {
	listener *listener
	callback sobek.Value
}

// _eventTarget is the JavaScript `EventTarget` constructor.
type _eventTarget struct{}

func (e *_eventTarget) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := e.prototype(rt)
	ctor := rt.ToValue(e.constructor).ToObject(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)
	_ = proto.SetSymbol(sobek.SymToStringTag, rt.ToValue("EventTarget"))
	return ctor, nil
}

func (*_eventTarget) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	target := newEventTarget(rt, call.This.ToObject(rt).Prototype())
	return target.ValueOf()
}

func (e *_eventTarget) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("addEventListener", e.addEventListener)
	_ = p.Set("removeEventListener", e.removeEventListener)
	_ = p.Set("dispatchEvent", e.dispatchEvent)
	return p
}

func (*_eventTarget) addEventListener(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(common.NewTypeErrorf(rt, common.ErrMissingArgs, `The "listener" argument must be specified`))
	}

	target := toEventTarget(rt, call.This)
	typ := call.Argument(0).String()
	opts := parseListenerOptions(call.Argument(2), false)
	l := newListener(rt, call.Argument(1), opts)
	if l == nil {
		return sobek.Undefined()
	}
	if target.contains(typ, l) {
		return sobek.Undefined()
	}
	if opts.signal != nil {
		if signalOptionHandler == nil {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType, `The "signal" argument must be an instance of AbortSignal`))
		}
		if !signalOptionHandler(rt, opts.signal, func() { target.remove(typ, l) }) {
			return sobek.Undefined()
		}
	}
	target.add(typ, l)
	return sobek.Undefined()
}

func (*_eventTarget) removeEventListener(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(common.NewTypeErrorf(rt, common.ErrMissingArgs, `The "listener" argument must be specified`))
	}

	target := toEventTarget(rt, call.This)
	typ := call.Argument(0).String()
	callback := call.Argument(1)
	if !validateListener(rt, callback) {
		return sobek.Undefined()
	}
	opts := parseListenerOptions(call.Argument(2), true)
	target.removeMatching(typ, callback, opts.capture)
	return sobek.Undefined()
}

func (*_eventTarget) dispatchEvent(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 1 {
		panic(common.NewTypeErrorf(rt, common.ErrMissingArgs, `The "event" argument must be specified`))
	}

	target := toEventTarget(rt, call.This)
	evt, ok := EventOf[Event](call.Argument(0))
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType, `The "event" argument must be an instance of Event. Received type %s`, call.Argument(0)))
	}
	return rt.ToValue(DispatchEvent(rt, target, evt))
}

func newEventTarget(rt *sobek.Runtime, proto *sobek.Object) *eventTarget {
	if proto == nil {
		proto = types.Prototype(rt, "EventTarget")
	}
	target := &eventTarget{
		rt:        rt,
		listeners: make(map[string][]*listener),
		handlers:  make(map[string]handler),
	}
	// A plain object so that JavaScript subclasses can add their own state.
	target.this = rt.NewObject()
	_ = target.this.SetPrototype(proto)
	_ = target.this.SetSymbol(symEventTarget, target)
	return target
}

type eventTarget struct {
	rt   *sobek.Runtime
	this *sobek.Object

	listeners map[string][]*listener
	handlers  map[string]handler
}

// ValueOf returns the JavaScript object for this target, creating and caching
// it on first use so that the same EventTarget always maps to the same
// JavaScript object.
func (t *eventTarget) ValueOf() *sobek.Object { return t.this }

func (t *eventTarget) EventHandler(typ string) sobek.Value {
	if h, ok := t.handlers[typ]; ok {
		return h.callback
	}
	return sobek.Null()
}

func (t *eventTarget) SetEventHandler(typ string, callback sobek.Value) {
	if h, ok := t.handlers[typ]; ok {
		t.remove(typ, h.listener)
		delete(t.handlers, typ)
	}
	if types.IsNil(callback) {
		return
	}
	l := newListener(t.rt, callback, listenerOptions{})
	if l == nil {
		return
	}
	t.add(typ, l)
	t.handlers[typ] = handler{listener: l, callback: callback}
}

func (t *eventTarget) contains(typ string, l *listener) bool {
	for _, existing := range t.listeners[typ] {
		if existing.matches(l.callback, l.capture) {
			return true
		}
	}
	return false
}

func (t *eventTarget) add(typ string, l *listener) {
	t.listeners[typ] = append(t.listeners[typ], l)
}

func (t *eventTarget) remove(typ string, l *listener) {
	l.removed.Store(true)
	listeners := t.listeners[typ]
	for i, existing := range listeners {
		if existing == l {
			t.listeners[typ] = append(listeners[:i:i], listeners[i+1:]...)
			return
		}
	}
}

func (t *eventTarget) removeMatching(typ string, callback sobek.Value, capture bool) {
	listeners := t.listeners[typ]
	for i, existing := range listeners {
		if existing.matches(callback, capture) {
			existing.removed.Store(true)
			t.listeners[typ] = append(listeners[:i:i], listeners[i+1:]...)
			return
		}
	}
}

func (t *eventTarget) invoke(evt *event) {
	listeners := t.listeners[evt._type]
	if len(listeners) == 0 {
		return
	}
	// Clone the listener list: listeners added while this target is invoked do
	// not run for this invocation.
	snapshot := make([]*listener, len(listeners))
	copy(snapshot, listeners)

	for _, l := range snapshot {
		if l.removed.Load() {
			continue
		}
		if evt.immediatePropagationStopped {
			return
		}
		if l.once {
			t.remove(evt._type, l)
		}
		invokeListener(t.rt, l, evt, t.ValueOf())
	}
}

// invokeListener invokes the listener with the current target as `this`.
// https://dom.spec.whatwg.org/#concept-event-listener-invoke
func invokeListener(rt *sobek.Runtime, l *listener, evt *event, currentTarget sobek.Value) {
	if l.passive {
		evt.inPassiveListener = true
		defer func() { evt.inPassiveListener = false }()
	}

	if callable, ok := sobek.AssertFunction(l.callback); ok {
		if _, err := callable(currentTarget, evt.this); err != nil {
			reportError(err)
		}
		return
	}

	obj, ok := l.callback.(*sobek.Object)
	if !ok {
		return
	}
	handleEvent := obj.Get("handleEvent")
	if types.IsNil(handleEvent) {
		return
	}
	callable, ok := sobek.AssertFunction(handleEvent)
	if !ok {
		// Node.js calls handleEvent unconditionally and reports the TypeError.
		reportError(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "listener" argument must be of type function or an object. Received type %s`,
			types.ValueType(handleEvent)))
		return
	}
	if _, err := callable(obj, evt.this); err != nil {
		reportError(err)
	}
}

func reportError(err any) {
	slog.Error("Uncaught Error", "error", err)
}

type listenerOptions struct {
	capture bool
	once    bool
	passive bool
	signal  sobek.Value
}

func parseListenerOptions(options sobek.Value, remove bool) listenerOptions {
	var opts listenerOptions
	if types.IsNil(options) {
		return opts
	}
	obj, ok := options.(*sobek.Object)
	if !ok {
		opts.capture = options.ToBoolean()
		return opts
	}
	if v := obj.Get("capture"); v != nil {
		opts.capture = v.ToBoolean()
	}
	if remove {
		return opts
	}
	if v := obj.Get("once"); v != nil {
		opts.once = v.ToBoolean()
	}
	if v := obj.Get("passive"); v != nil {
		opts.passive = v.ToBoolean()
	}
	if v := obj.Get("signal"); !types.IsNil(v) {
		opts.signal = v
	}
	return opts
}

// validateListener checks the listener argument of addEventListener and
// removeEventListener. It returns false when the listener has no effect, and
// panics with ERR_INVALID_ARG_TYPE otherwise, matching Node.js.
func validateListener(rt *sobek.Runtime, value sobek.Value) bool {
	if types.IsNil(value) {
		// Node.js warns and ignores null/undefined listeners.
		return false
	}
	// Functions are objects as well.
	if _, ok := value.(*sobek.Object); ok {
		return true
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`The "listener" argument must be of type function or an object. Received type %s`,
		types.ValueType(value)))
}

func newListener(rt *sobek.Runtime, callback sobek.Value, opts listenerOptions) *listener {
	if !validateListener(rt, callback) {
		return nil
	}
	return &listener{
		callback: callback,
		capture:  opts.capture,
		once:     opts.once,
		passive:  opts.passive,
	}
}

var symEventTarget = sobek.NewSymbol("Symbol.EventTarget")

// EventTargetOf returns the EventTarget behind the value. The JavaScript object
// of a Go defined target is branded with a pointer to the value, so a subclass
// is looked up with its pointer type, e.g. EventTargetOf[*MyTarget](value),
// while EventTargetOf[EventTarget](value) returns any target.
func EventTargetOf[T EventTarget](value sobek.Value) (ret T, _ bool) {
	o, ok := value.(*sobek.Object)
	if !ok {
		return
	}
	brand := o.GetSymbol(symEventTarget)
	if brand == nil {
		return
	}
	target, ok := brand.Export().(T)
	return target, ok
}

func toEventTarget(rt *sobek.Runtime, value sobek.Value) EventTarget {
	if target, ok := EventTargetOf[EventTarget](value); ok {
		return target
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidThis, `Value of "this" must be of type EventTarget`))
}
