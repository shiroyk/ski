package events

import (
	"fmt"
	"reflect"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/common"
)

// EventInit is the Go representation of the DOM EventInit dictionary.
// https://dom.spec.whatwg.org/#dictdef-eventinit
type EventInit struct {
	Bubbles    bool
	Cancelable bool
	Composed   bool
}

// Event defines the DOM Event interface.
// The exported method is the event source facing API, the unexported one keeps
// the interface sealed: a type outside this package can only satisfy it by
// embedding a real Event instance.
// https://dom.spec.whatwg.org/#interface-event
type Event interface {
	ValueOf() *sobek.Object
	base() *event
}

// NewEvent creates a new Event.
func NewEvent(rt *sobek.Runtime, typ string, init EventInit) Event {
	evt := newEvent(typ, init)
	evt.this = newEventObject(rt, evt, nil)
	return evt
}

// NewEventOf creates a new Event of type E, proto nil means use the Event
// prototype. E must embed an Event field, it is set to the new event so a Go
// defined subclass can hold its own state, and the JavaScript object is branded
// with *E, so EventOf[*E](value) returns it.
func NewEventOf[E Event](rt *sobek.Runtime, typ string, init EventInit, proto *sobek.Object) *E {
	var e E
	brand, ok := any(&e).(Event)
	if !ok {
		panic(fmt.Sprintf("events: %T must implement Event", &e))
	}
	base := newEvent(typ, init)
	base.this = newEventObject(rt, brand, proto)
	holder := reflect.ValueOf(&e).Elem().FieldByName("Event")
	if !holder.IsValid() || !holder.CanSet() {
		panic(fmt.Sprintf("events: %T must embed an events.Event field", e))
	}
	holder.Set(reflect.ValueOf(Event(base)))
	return &e
}

// EventOf returns the Event behind the value. The JavaScript object of a Go
// defined event is branded with a pointer to the value, so a subclass is looked
// up with its pointer type, e.g. EventOf[*MyEvent](value), while
// EventOf[Event](value) returns any event.
func EventOf[E Event](value sobek.Value) (ret E, _ bool) {
	o, ok := value.(*sobek.Object)
	if !ok {
		return
	}
	brand := o.GetSymbol(symEvent)
	if brand == nil {
		return
	}
	evt, ok := brand.Export().(E)
	return evt, ok
}

type eventPhase int

const (
	eventPhaseNone eventPhase = iota
	eventPhaseCapturing
	eventPhaseAtTarget
	eventPhaseBubbling
)

// _event is the JavaScript `Event` constructor.
type _event struct{}

func (e *_event) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := e.prototype(rt)
	ctor := rt.ToValue(e.constructor).ToObject(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)
	setEventPhaseConstants(ctor)
	_ = proto.SetSymbol(sobek.SymToStringTag, rt.ToValue("Event"))
	return ctor, nil
}

func setEventPhaseConstants(o *sobek.Object) {
	_ = o.Set("NONE", int(eventPhaseNone))
	_ = o.Set("CAPTURING_PHASE", int(eventPhaseCapturing))
	_ = o.Set("AT_TARGET", int(eventPhaseAtTarget))
	_ = o.Set("BUBBLING_PHASE", int(eventPhaseBubbling))
}

func (*_event) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) < 1 {
		panic(common.NewTypeErrorf(rt, common.ErrMissingArgs, `The "type" argument must be specified`))
	}

	init := common.ExportDict[EventInit](rt, call.Argument(1), "options")
	evt := newEvent(call.Argument(0).String(), init)
	evt.this = newEventObject(rt, evt, call.This.ToObject(rt).Prototype())
	return evt.this
}

func (e *_event) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()

	// Properties
	_ = p.DefineAccessorProperty("type", rt.ToValue(e.type_), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("target", rt.ToValue(e.target), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("currentTarget", rt.ToValue(e.currentTarget), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("eventPhase", rt.ToValue(e.eventPhase), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("timeStamp", rt.ToValue(e.timeStamp), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("bubbles", rt.ToValue(e.bubbles), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("cancelable", rt.ToValue(e.cancelable), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("composed", rt.ToValue(e.composed), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("defaultPrevented", rt.ToValue(e.defaultPrevented), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("isTrusted", rt.ToValue(e.isTrusted), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	// Methods
	_ = p.Set("stopPropagation", e.stopPropagation)
	_ = p.Set("stopImmediatePropagation", e.stopImmediatePropagation)
	_ = p.Set("preventDefault", e.preventDefault)
	_ = p.Set("composedPath", e.composedPath)

	return p
}

func (*_event) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This)._type)
}

func (*_event) target(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	if evt.target == nil {
		return sobek.Null()
	}
	return evt.target.ValueOf()
}

func (*_event) currentTarget(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	if evt.currentTarget == nil {
		return sobek.Null()
	}
	return evt.currentTarget.ValueOf()
}

func (*_event) eventPhase(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(int(toEvent(rt, call.This).phase))
}

func (*_event) timeStamp(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This).timeStamp)
}

func (*_event) bubbles(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This).bubbles)
}

func (*_event) cancelable(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This).cancelable)
}

func (*_event) composed(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This).composed)
}

func (*_event) defaultPrevented(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toEvent(rt, call.This).defaultPrevented)
}

func (*_event) isTrusted(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(false)
}

func (*_event) stopPropagation(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toEvent(rt, call.This).propagationStopped = true
	return sobek.Undefined()
}

func (*_event) stopImmediatePropagation(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	evt.immediatePropagationStopped = true
	evt.propagationStopped = true
	return sobek.Undefined()
}

func (*_event) preventDefault(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	// A passive listener can not cancel the event.
	if evt.cancelable && !evt.inPassiveListener {
		evt.defaultPrevented = true
	}
	return sobek.Undefined()
}

func (*_event) composedPath(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toEvent(rt, call.This)
	path := make([]any, len(evt.path))
	for i, target := range evt.path {
		path[i] = target.ValueOf()
	}
	return rt.ToValue(path)
}

func toEvent(rt *sobek.Runtime, value sobek.Value) *event {
	if evt, ok := EventOf[Event](value); ok {
		return evt.base()
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidThis, `Value of "this" must be of type Event`))
}

// newEvent creates the base state of an event.
func newEvent(typ string, init EventInit) *event {
	return &event{
		_type:      typ,
		timeStamp:  time.Now().UnixNano() / int64(time.Millisecond),
		bubbles:    init.Bubbles,
		cancelable: init.Cancelable,
		composed:   init.Composed,
	}
}

var symEvent = sobek.NewSymbol("Symbol.Event")

// newEventObject creates the JavaScript object of an event. The object is a
// plain object so that listeners can share expando properties and JavaScript
// subclasses can add their own state; brand is stored under a symbol so the
// Event can be recovered without exporting the object.
func newEventObject(rt *sobek.Runtime, event Event, proto *sobek.Object) *sobek.Object {
	if proto == nil {
		proto = types.Prototype(rt, "Event")
	}
	obj := rt.NewObject()
	_ = obj.SetPrototype(proto)
	_ = obj.SetSymbol(symEvent, event)
	return obj
}

// event is the Go representation of an Event instance.
type event struct {
	this          *sobek.Object
	_type         string
	target        EventTarget
	currentTarget EventTarget
	path          []EventTarget
	phase         eventPhase
	timeStamp     int64

	bubbles    bool
	cancelable bool
	composed   bool

	defaultPrevented bool

	dispatching                 bool
	propagationStopped          bool
	immediatePropagationStopped bool
	inPassiveListener           bool
}

func (e *event) base() *event { return e }

func (e *event) ValueOf() *sobek.Object { return e.this }
