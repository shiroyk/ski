package dom

import (
	"context"
	"log/slog"
	"reflect"
	"slices"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules/signal"
)

// EventTarget defines the DOM EventTarget interface
// https://dom.spec.whatwg.org/#interface-eventtarget
type EventTarget interface {
	// AddEventListener registers an event handler of a specific event type on the EventTarget
	AddEventListener(typ string, listener EventListener)
	// RemoveEventListener removes an event listener from the EventTarget
	RemoveEventListener(typ string, listener EventListener)
	// DispatchEvent dispatches an event to this EventTarget
	DispatchEvent(event Event) bool
	// Listeners returns all event listeners for a specific event type
	Listeners(typ string) []EventListener

	toValue
	dispatchEvent(e Event)
	parent() EventTarget
	setParent(target EventTarget)
}

type EventListener interface {
	HandleEvent(event Event) error
	Equals(e EventListener) bool
	Options() AddEventListenerOptions
}

type EventListenerOptions struct {
	Capture bool
}

type AddEventListenerOptions struct {
	EventListenerOptions
	Passive bool
	Once    bool
	Signal  context.Context
}

type eventTarget struct{}

func (e *eventTarget) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ctor := rt.ToValue(e.constructor).ToObject(rt)
	p := e.prototype(rt)
	_ = p.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(p)
	_ = ctor.Set("prototype", p)
	return ctor, nil
}

func (e *eventTarget) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	target := NewEventTarget()
	ret := rt.NewObject()
	_ = ret.SetSymbol(symEventTarget, target)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

func (e *eventTarget) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()

	_ = p.Set("addEventListener", e.addEventListener)
	_ = p.Set("removeEventListener", e.removeEventListener)
	_ = p.Set("dispatchEvent", e.dispatchEvent)

	return p
}

func (*eventTarget) addEventListener(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("Failed to execute 'addEventListener': 2 arguments required, but only 0 present."))
	}

	this := toEventTarget(rt, call.This)
	typ := call.Argument(0).String()
	listener := call.Argument(1)
	options := call.Argument(2)

	if listener.ExportType() != typeFunc {
		return sobek.Undefined()
	}

	el := &jsEventListener{
		rt: rt,
		fn: listener,
	}

	if !sobek.IsUndefined(options) {
		if options.ExportType().Kind() == reflect.Bool {
			el.opts.Capture = options.ToBoolean()
		} else if obj := options.ToObject(rt); obj != nil {
			if v := obj.Get("capture"); v != nil {
				el.opts.Capture = v.ToBoolean()
			}
			if v := obj.Get("once"); v != nil {
				el.opts.Once = v.ToBoolean()
			}
			if v := obj.Get("passive"); v != nil {
				el.opts.Passive = v.ToBoolean()
			}
			if v := obj.Get("signal"); v != nil {
				el.opts.Signal = signal.Context(rt, v)
				enqueue := js.EnqueueJob(rt)
				context.AfterFunc(el.opts.Signal, func() {
					enqueue(func() error {
						this.RemoveEventListener(typ, el)
						return nil
					})
				})
			}
		}
	}

	this.AddEventListener(typ, el)

	return sobek.Undefined()
}

func (*eventTarget) removeEventListener(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("Failed to execute 'removeEventListener': 2 arguments required, but only 0 present."))
	}

	this := toEventTarget(rt, call.This)
	typ := call.Argument(0).String()
	listener := call.Argument(1)
	options := call.Argument(2)

	if listener.ExportType() != typeFunc {
		return sobek.Undefined()
	}

	el := &jsEventListener{
		rt: rt,
		fn: listener,
	}

	if !sobek.IsUndefined(options) {
		if obj := options.ToObject(rt); obj != nil {
			if v := obj.Get("capture"); v != nil {
				el.opts.Capture = v.ToBoolean()
			}
		}
	}

	this.RemoveEventListener(typ, el)

	return sobek.Undefined()
}

func (*eventTarget) dispatchEvent(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("Failed to execute 'dispatchEvent': 1 argument required, but only 0 present."))
	}

	this := toEventTarget(rt, call.This)
	return rt.ToValue(this.DispatchEvent(toEvent(rt, call.Argument(0), true)))
}

// NewEventTarget creates a new EventTarget instance
func NewEventTarget() EventTarget {
	return &_eventTarget{
		listeners: make(map[string][]EventListener),
	}
}

var (
	symEventTarget = sobek.NewSymbol("Symbol.EventTarget")
	typeFunc       = reflect.TypeOf((func(sobek.FunctionCall) sobek.Value)(nil))
)

func toEventTarget(rt *sobek.Runtime, value sobek.Value, isValue ...bool) EventTarget {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symEventTarget); v != nil {
			return v.Export().(EventTarget)
		}
	}
	if len(isValue) > 0 && isValue[0] {
		panic(rt.NewTypeError(`Value must be of type EventTarget`))
	}
	panic(rt.NewTypeError(`Value of "this" must be of type EventTarget`))
}

type _eventTarget struct {
	parentTarget EventTarget
	listeners    map[string][]EventListener
}

func (t *_eventTarget) toValue(this sobek.Value, rt *sobek.Runtime) sobek.Value {
	if this == nil {
		this = rt.Get("EventTarget")
	}
	if this == nil {
		panic(rt.NewTypeError("EventTarget is not defined"))
	}
	ret := rt.NewObject()
	_ = ret.SetSymbol(symEventTarget, t)
	_ = ret.SetPrototype(this.ToObject(rt).Prototype())
	return ret
}

func (t *_eventTarget) AddEventListener(typ string, listener EventListener) {
	t.listeners[typ] = append(t.listeners[typ], listener)
}

func (t *_eventTarget) RemoveEventListener(typ string, listener EventListener) {
	listeners := t.listeners[typ]
	t.listeners[typ] = slices.DeleteFunc(listeners, func(l EventListener) bool { return l.Equals(listener) })
}

func (t *_eventTarget) DispatchEvent(e Event) bool {
	defer e.reset()
	e.setTarget(t)

	// Capture phase
	e.setEventPhase(EventPhaseCapturing)
	parents := make([]EventTarget, 0)
	for p := t.parentTarget; p != nil; p = p.parent() {
		parents = append(parents, p)
	}
	// Dispatch in reverse order (from root to target's parent)
	for i := len(parents) - 1; i >= 0; i-- {
		if e.isPropagationStopped() {
			break
		}
		parents[i].dispatchEvent(e)
	}

	// Target phase
	e.setEventPhase(EventPhaseAtTarget)
	t.dispatchEvent(e)

	// Bubble phase
	if e.Bubbles() {
		e.setEventPhase(EventPhaseBubbling)
		if t.parentTarget != nil && !e.isPropagationStopped() {
			t.parentTarget.dispatchEvent(e)
		}
	}

	return !e.DefaultPrevented()
}

func (t *_eventTarget) dispatchEvent(e Event) {
	e.setCurrentTarget(t)
	capture := e.EventPhase() == EventPhaseCapturing
	for _, listener := range t.Listeners(e.Type()) {
		if listener.Options().Capture && !capture || !listener.Options().Capture && capture {
			continue
		}

		if e.isImmediatePropagationStopped() {
			break
		}

		if listener.Options().Once {
			t.RemoveEventListener(e.Type(), listener)
		}

		if !listener.Options().Passive {
			if err := listener.HandleEvent(e); err != nil {
				slog.Error("Uncaught Error", "error", err)
			}
		} else {
			prevDefaultPrevented := e.DefaultPrevented()
			if err := listener.HandleEvent(e); err != nil {
				slog.Error("Uncaught Error", "error", err)
			}
			if e.DefaultPrevented() && !prevDefaultPrevented {
				e.setDefaultPrevented(false)
			}
		}
	}

	if e.EventPhase() == EventPhaseBubbling {
		if t.parentTarget != nil && !e.isPropagationStopped() {
			t.parentTarget.dispatchEvent(e)
		}
	}
}

func (t *_eventTarget) Listeners(typ string) []EventListener {
	return append([]EventListener(nil), t.listeners[typ]...)
}

func (t *_eventTarget) parent() EventTarget { return t.parentTarget }

func (t *_eventTarget) setParent(target EventTarget) { t.parentTarget = target }

type jsEventListener struct {
	rt   *sobek.Runtime
	fn   sobek.Value
	opts AddEventListenerOptions
}

func (j *jsEventListener) Options() AddEventListenerOptions { return j.opts }

func (j *jsEventListener) HandleEvent(event Event) error {
	callable, ok := sobek.AssertFunction(j.fn)
	if !ok {
		return nil
	}
	var value sobek.Value
	ex := j.rt.Try(func() { value = event.toValue(nil, j.rt) })
	if ex != nil {
		return j.log(ex)
	}
	_, err := callable(sobek.Undefined(), value)
	if err != nil {
		return j.log(err)
	}
	return nil
}

func (j *jsEventListener) log(err error) error {
	console := j.rt.Get("console")
	if console != nil {
		out, ok := sobek.AssertFunction(console.(*sobek.Object).Get("error"))
		if ok {
			_, err = out(sobek.Undefined(), j.rt.ToValue(err.Error()))
			return err
		}
	}
	return err
}

func (j *jsEventListener) Equals(e EventListener) bool {
	other, ok := e.(*jsEventListener)
	if !ok {
		return false
	}
	return j.fn.StrictEquals(other.fn)
}

// NewEventListener creates a new EventListener
func NewEventListener(fn func(Event) error, opts AddEventListenerOptions) EventListener {
	return &goEventListener{
		fn:   fn,
		opts: opts,
		id:   newID(),
	}
}

type goEventListener struct {
	fn   func(Event) error
	opts AddEventListenerOptions
	id   uint32
}

func (j *goEventListener) Options() AddEventListenerOptions { return j.opts }
func (j *goEventListener) HandleEvent(event Event) error    { return j.fn(event) }
func (j *goEventListener) Equals(e EventListener) bool {
	other, ok := e.(*goEventListener)
	if !ok {
		return false
	}
	return j.id == other.id
}
