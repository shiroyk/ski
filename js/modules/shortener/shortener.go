package shortener

import (
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/internal/di"
	"github.com/shiroyk/cloudcat/js/modules"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Shortener{di.MustResolve[cache.Shortener]()}
}

// Global returns is it is a global module
func (*Module) Global() bool {
	return false
}

func init() {
	modules.Register("shortener", &Module{})
}

// Shortener is URL shortener to reduce a long link and headers.
type Shortener struct {
	shortener cache.Shortener
}

// Set returns to shorten identifier for the given HTTP request.
func (s *Shortener) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(s.shortener.Set(call.Argument(0).String(), time.Duration(call.Argument(1).ToInteger())))
}

// Get returns the HTTP request for the given identifier.
func (s *Shortener) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if h, ok := s.shortener.Get(call.Argument(0).String()); ok {
		return vm.ToValue(h)
	}
	return
}
