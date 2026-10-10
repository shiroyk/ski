package events_test

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/events"
	"github.com/stretchr/testify/assert"
)

// externalEvent proves a type of another package satisfies the sealed
// events.Event interface by embedding it, the same way AbortSignal embeds
// events.EventTarget.
type externalEvent struct {
	events.Event
	data string
}

var _ events.Event = (*externalEvent)(nil)

// externalTarget proves a type of another package satisfies the sealed
// events.EventTarget interface by embedding it.
type externalTarget struct {
	events.EventTarget
	hits int
}

var _ events.EventTarget = (*externalTarget)(nil)

func TestExternalEventTarget(t *testing.T) {
	vm := modulestest.New()
	rt := vm.Runtime()
	// A subclass prototype inherits the EventTarget prototype, like
	// AbortSignal.prototype does.
	proto := rt.NewObject()
	_ = proto.SetPrototype(types.Prototype(rt, "EventTarget"))

	target := events.NewEventTargetOf[externalTarget](rt, proto)
	target.SetEventHandler("external", rt.ToValue(func(sobek.FunctionCall) sobek.Value {
		target.hits++
		return sobek.Undefined()
	}))

	// The JavaScript object uses the given prototype and resolves back to the
	// most derived Go value.
	assert.True(t, target.ValueOf().Prototype() == proto)
	eventTarget, ok := events.EventTargetOf[*externalTarget](target.ValueOf())
	assert.True(t, ok)
	assert.Equal(t, target, eventTarget)

	events.DispatchEvent(rt, target, events.NewEvent(rt, "external", events.EventInit{}))
	assert.Equal(t, 1, target.hits)

	// Dispatching from JavaScript hits the same handler.
	_, err := vm.RunModule(context.Background(), `
		export default (target) => {
			target.dispatchEvent(new Event('external'));
		}
	`, target.ValueOf())
	assert.NoError(t, err)
	assert.Equal(t, 2, target.hits)
}

func TestExternalEvent(t *testing.T) {
	vm := modulestest.New()
	rt := vm.Runtime()
	target := events.NewEventTarget(rt)

	p := rt.NewObject()
	event := events.NewEventOf[externalEvent](rt, "external", events.EventInit{}, p)
	event.data = "test"
	_ = p.DefineAccessorProperty("data", rt.ToValue(func() string { return event.data }), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	var callback bool
	target.SetEventHandler("external", rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
		callback = assert.Equal(t, event.data, call.Argument(0).ToObject(rt).Get("data").String())
		return sobek.Undefined()
	}))

	events.DispatchEvent(rt, target, event)
	assert.True(t, callback)
}
