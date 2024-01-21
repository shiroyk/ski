package http

import (
	"context"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski/js"
)

// abortController interface represents a controller object
// that allows you to abort one or more Web requests as and when desired.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortController.
type abortController struct {
	Signal  *abortSignal
	Aborted bool
	Reason  string
}

func (c *abortController) Abort() {
	c.Signal.abort()
	c.Aborted = c.Signal.Aborted
	c.Reason = c.Signal.Reason
}

// AbortController Constructor
type AbortController struct{}

// Instantiate module
func (*AbortController) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	return rt.ToValue(func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		signal := new(abortSignal)
		signal.ctx, signal.cancel = context.WithCancel(js.Context(vm))
		return vm.ToValue(&abortController{Signal: signal}).ToObject(vm)
	}), nil
}

// Global it is a global module
func (*AbortController) Global() {}

// abortSignal represents a signal object that allows you to communicate
// with http request and abort it.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
type abortSignal struct {
	ctx     context.Context
	cancel  context.CancelFunc
	once    sync.Once
	Aborted bool
	Reason  string
}

func (s *abortSignal) abort() {
	s.once.Do(func() {
		s.Aborted = true
		s.cancel()
		if err := s.ctx.Err(); err != nil {
			s.Reason = err.Error()
		}
	})
}

type AbortSignal struct{}

func (*AbortSignal) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	object := rt.NewObject()
	_ = object.Set("abort", func(_ goja.FunctionCall) goja.Value {
		signal := new(abortSignal)
		signal.ctx, signal.cancel = context.WithCancel(context.Background())
		signal.abort()
		return rt.ToValue(signal).ToObject(rt)
	})
	_ = object.Set("timeout", func(call goja.FunctionCall) goja.Value {
		timeout := call.Argument(0).ToInteger()
		signal := new(abortSignal)
		signal.ctx, signal.cancel = context.WithTimeout(js.Context(rt), time.Duration(timeout))
		return rt.ToValue(signal).ToObject(rt)
	})
	return object, nil
}

func (*AbortSignal) Global() {}
