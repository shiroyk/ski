package http

import (
	"context"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js"
)

// AbortController interface represents a controller object
// that allows you to abort one or more Web requests as and when desired.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortController.
type AbortController struct {
	Signal  *AbortSignal
	Aborted bool
	Reason  string
}

func (c *AbortController) Abort() {
	c.Signal.abort()
	c.Aborted = c.Signal.Aborted
	c.Reason = c.Signal.Reason
}

// AbortControllerConstructor AbortController Constructor
type AbortControllerConstructor struct{}

// Exports AbortController Constructor
func (*AbortControllerConstructor) Exports() any {
	return func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		signal := new(AbortSignal)
		parent := js.VMContext(vm)
		signal.ctx, signal.cancel = context.WithCancel(parent)
		return vm.ToValue(&AbortController{Signal: signal}).ToObject(vm)
	}
}

// Global it is a global module
func (*AbortControllerConstructor) Global() {}

// AbortSignal represents a signal object that allows you to communicate
// with http request and abort it.
// https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
type AbortSignal struct {
	ctx     context.Context
	cancel  context.CancelFunc
	Aborted bool
	Reason  string
}

func (s *AbortSignal) abort() {
	if s.Aborted {
		return
	}
	s.Aborted = true
	s.cancel()
	if err := s.ctx.Err(); err != nil {
		s.Reason = err.Error()
	}
}

type AbortSignalModule struct{}

func (*AbortSignalModule) Exports() any { return new(abortSignalInstance) }

func (*AbortSignalModule) Global() {}

type abortSignalInstance struct{}

func (s *abortSignalInstance) Abort(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	signal := new(AbortSignal)
	signal.ctx, signal.cancel = context.WithCancel(context.Background())
	signal.abort()
	return vm.ToValue(signal).ToObject(vm)
}

func (s *abortSignalInstance) Timeout(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	timeout := call.Argument(0).ToInteger()
	signal := new(AbortSignal)
	signal.ctx, signal.cancel = context.WithTimeout(js.VMContext(vm), time.Duration(timeout))
	return vm.ToValue(signal).ToObject(vm)
}
