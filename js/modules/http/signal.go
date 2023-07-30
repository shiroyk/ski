package http

import (
	"context"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js"
)

// AbortSignal represents a signal object that allows you to communicate
// with http request and abort it.
type AbortSignal struct {
	ctx                   context.Context
	timeoutCancel, cancel context.CancelFunc
	Aborted               bool
	Reason                string
}

// AbortSignalConstructor Signal Constructor
type AbortSignalConstructor struct{}

// Exports instance AbortSignal Constructor
func (*AbortSignalConstructor) Exports() any {
	return func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		timeout := call.Argument(0).ToInteger()
		signal := new(AbortSignal)
		parent := js.VMContext(vm)
		if timeout > 0 {
			parent, signal.timeoutCancel = context.WithTimeout(parent, time.Duration(timeout))
		}
		signal.ctx, signal.cancel = context.WithCancel(parent)
		return vm.ToValue(signal).ToObject(vm)
	}
}

// Global it is a global module
func (*AbortSignalConstructor) Global() {}

// Abort the signal
func (s *AbortSignal) Abort() {
	if s.Aborted {
		return
	}
	s.Aborted = true
	s.cancel()
	if err := s.ctx.Err(); err != nil {
		s.Reason = err.Error()
	}
}

func (s *AbortSignal) timeout() {
	s.Aborted = true
	if s.timeoutCancel != nil {
		s.timeoutCancel()
	}
}
