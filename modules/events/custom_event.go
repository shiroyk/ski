package events

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/common"
)

// CustomEventInit is the Go representation of the DOM CustomEventInit dictionary.
// https://dom.spec.whatwg.org/#dictdef-customeventinit
type CustomEventInit struct {
	EventInit
	Detail sobek.Value
}

// NewCustomEvent creates a new CustomEvent.
func NewCustomEvent(rt *sobek.Runtime, typ string, init CustomEventInit) Event {
	return newCustomEvent(rt, typ, init, types.Prototype(rt, "CustomEvent"))
}

// customEvent is the JavaScript `CustomEvent` constructor.
type customEvent struct{}

func (c *customEvent) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := c.prototype(rt)
	ctor := rt.ToValue(c.constructor).ToObject(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)
	// CustomEvent.prototype inherits from Event.prototype.
	_ = proto.SetPrototype(types.Prototype(rt, "Event"))
	_ = proto.SetSymbol(sobek.SymToStringTag, rt.ToValue("CustomEvent"))
	return ctor, nil
}

func (c *customEvent) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) < 1 {
		panic(common.NewTypeErrorf(rt, common.ErrMissingArgs, `The "type" argument must be specified`))
	}

	init := common.ExportDict[CustomEventInit](rt, call.Argument(1), "options")
	evt := newCustomEvent(rt, call.Argument(0).String(), init, call.This.ToObject(rt).Prototype())
	return evt.ValueOf()
}

func (c *customEvent) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("detail", rt.ToValue(c.detail), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	return p
}

// detail returns the detail data of the _event, null by default.
func (*customEvent) detail(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	evt := toCustomEvent(rt, call.This)
	if types.IsNil(evt.detail) {
		return sobek.Null()
	}
	return evt.detail
}

// _customEvent is the Go representation of a CustomEvent instance.
type _customEvent struct {
	Event
	detail sobek.Value
}

func newCustomEvent(rt *sobek.Runtime, typ string, init CustomEventInit, proto *sobek.Object) *_customEvent {
	evt := NewEventOf[_customEvent](rt, typ, init.EventInit, proto)
	evt.detail = init.Detail
	return evt
}

func toCustomEvent(rt *sobek.Runtime, value sobek.Value) *_customEvent {
	evt, ok := EventOf[*_customEvent](value)
	if ok {
		return evt
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidThis, `Value of "this" must be of type CustomEvent`))
}
