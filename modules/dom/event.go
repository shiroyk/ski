package dom

import (
	"time"

	"github.com/grafana/sobek"
)

// Event defines the DOM Event interface
// https://dom.spec.whatwg.org/#event
type Event interface {
	Type() string
	Target() EventTarget
	CurrentTarget() EventTarget
	EventPhase() EventPhase
	TimeStamp() int64
	Bubbles() bool
	Cancelable() bool
	DefaultPrevented() bool

	StopPropagation()
	StopImmediatePropagation()
	PreventDefault()

	toValue
	reset()
	setTarget(target EventTarget)
	setCurrentTarget(target EventTarget)
	setEventPhase(eventPhase EventPhase)
	setBubbles(bubbles bool)
	setCancelable(cancelable bool)
	setDefaultPrevented(defaultPrevented bool)
	isPropagationStopped() bool
	isImmediatePropagationStopped() bool
}

type EventPhase int

const (
	EventPhaseNone EventPhase = iota
	EventPhaseCapturing
	EventPhaseAtTarget
	EventPhaseBubbling
)

type event struct{}

func (e *event) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ctor := rt.ToValue(e.constructor).ToObject(rt)
	p := e.prototype(rt)
	_ = p.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(p)
	_ = ctor.Set("prototype", p)
	_ = ctor.Set("NONE", EventPhaseNone)
	_ = ctor.Set("CAPTURING_PHASE", EventPhaseCapturing)
	_ = ctor.Set("AT_TARGET", EventPhaseAtTarget)
	_ = ctor.Set("BUBBLING_PHASE", EventPhaseBubbling)
	return ctor, nil
}

func (e *event) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("Failed to construct 'Event': 1 argument required, but only 0 present."))
	}

	eventType := call.Argument(0).String()
	options := call.Argument(1)

	evt := NewEvent(eventType)

	if !sobek.IsUndefined(options) {
		if obj := options.ToObject(rt); obj != nil {
			if v := obj.Get("bubbles"); v != nil {
				evt.setBubbles(v.ToBoolean())
			}
			if v := obj.Get("cancelable"); v != nil {
				evt.setCancelable(v.ToBoolean())
			}
		}
	}

	ret := rt.NewObject()
	_ = ret.SetSymbol(symEvent, evt)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

func (e *event) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()

	// Properties
	_ = p.DefineAccessorProperty("type", rt.ToValue(e.type_), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("target", rt.ToValue(e.target), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("currentTarget", rt.ToValue(e.currentTarget), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("eventPhase", rt.ToValue(e.eventPhase), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("timeStamp", rt.ToValue(e.timeStamp), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("bubbles", rt.ToValue(e.bubbles), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("cancelable", rt.ToValue(e.cancelable), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("defaultPrevented", rt.ToValue(e.defaultPrevented), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	// Methods
	_ = p.Set("stopPropagation", e.stopPropagation)
	_ = p.Set("stopImmediatePropagation", e.stopImmediatePropagation)
	_ = p.Set("preventDefault", e.preventDefault)

	return p
}

func (*event) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.Type())
}

func (*event) target(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	if target := evt.Target(); target != nil {
		return rt.ToValue(target.toValue(nil, rt))
	}
	return sobek.Null()
}

func (*event) currentTarget(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	if target := evt.CurrentTarget(); target != nil {
		return rt.ToValue(target.toValue(nil, rt))
	}
	return sobek.Null()
}

func (*event) eventPhase(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.EventPhase())
}

func (*event) timeStamp(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.TimeStamp())
}

func (*event) bubbles(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.Bubbles())
}

func (*event) cancelable(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.Cancelable())
}

func (*event) defaultPrevented(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	return rt.ToValue(evt.DefaultPrevented())
}

func (*event) stopPropagation(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	evt.StopPropagation()
	return sobek.Undefined()
}

func (*event) stopImmediatePropagation(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	evt.StopImmediatePropagation()
	return sobek.Undefined()
}

func (*event) preventDefault(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	evt.PreventDefault()
	return sobek.Undefined()
}

var symEvent = sobek.NewSymbol("Symbol.Event")

func toEvent(rt *sobek.Runtime, value sobek.Value, isValue ...bool) Event {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symEvent); v != nil {
			return v.Export().(Event)
		}
	}
	if len(isValue) > 0 && isValue[0] {
		panic(rt.NewTypeError(`Value must be of type Event`))
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Event`))
}

// NewEvent creates a new Event instance
func NewEvent(typ string) Event {
	return &_event{
		typ:       typ,
		timeStamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

type _event struct {
	typ                         string
	target                      EventTarget
	currentTarget               EventTarget
	eventPhase                  EventPhase
	timeStamp                   int64
	bubbles                     bool
	cancelable                  bool
	defaultPrevented            bool
	propagationStopped          bool
	immediatePropagationStopped bool
}

func (e *_event) Type() string                              { return e.typ }
func (e *_event) Target() EventTarget                       { return e.target }
func (e *_event) CurrentTarget() EventTarget                { return e.currentTarget }
func (e *_event) EventPhase() EventPhase                    { return e.eventPhase }
func (e *_event) TimeStamp() int64                          { return e.timeStamp }
func (e *_event) Bubbles() bool                             { return e.bubbles }
func (e *_event) Cancelable() bool                          { return e.cancelable }
func (e *_event) DefaultPrevented() bool                    { return e.defaultPrevented }
func (e *_event) StopPropagation()                          { e.propagationStopped = true }
func (e *_event) setTarget(target EventTarget)              { e.target = target }
func (e *_event) setCurrentTarget(target EventTarget)       { e.currentTarget = target }
func (e *_event) setEventPhase(eventPhase EventPhase)       { e.eventPhase = eventPhase }
func (e *_event) setBubbles(bubbles bool)                   { e.bubbles = bubbles }
func (e *_event) setCancelable(cancelable bool)             { e.cancelable = cancelable }
func (e *_event) setDefaultPrevented(defaultPrevented bool) { e.defaultPrevented = defaultPrevented }
func (e *_event) isPropagationStopped() bool                { return e.propagationStopped }
func (e *_event) isImmediatePropagationStopped() bool       { return e.immediatePropagationStopped }

func (e *_event) StopImmediatePropagation() {
	e.immediatePropagationStopped = true
	e.propagationStopped = true
}

func (e *_event) PreventDefault() {
	if e.cancelable {
		e.defaultPrevented = true
	}
}

func (e *_event) toValue(this sobek.Value, rt *sobek.Runtime) sobek.Value {
	if this == nil {
		this = rt.Get("Event")
	}
	if this == nil {
		panic(rt.NewTypeError("Event is not defined"))
	}
	ret := rt.NewObject()
	_ = ret.SetSymbol(symEvent, e)
	_ = ret.SetPrototype(this.ToObject(rt).Prototype())
	return ret
}

func (e *_event) reset() {
	e.target = nil
	e.currentTarget = nil
	e.eventPhase = EventPhaseNone
}
