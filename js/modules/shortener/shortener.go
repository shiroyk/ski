package shortener

import (
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/modules"
)

type Module struct{}

func (*Module) Exports() any {
	return &Shortener{di.MustResolve[cache.Shortener]()}
}

func init() {
	modules.Register("shortener", &Module{})
}

type Shortener struct {
	shortener cache.Shortener
}

func (s *Shortener) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(s.shortener.Set(call.Argument(0).String(), time.Duration(call.Argument(1).ToInteger())))
}

func (s *Shortener) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if h, ok := s.shortener.Get(call.Argument(0).String()); ok {
		return vm.ToValue(h)
	}
	return
}
