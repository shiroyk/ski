package js

import (
	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
)

type jsShortener struct {
	shortener cache.Shortener
}

func (s *jsShortener) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(s.shortener.Set(call.Argument(0).String()))
}

func (s *jsShortener) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if h, ok := s.shortener.Get(call.Argument(0).String()); ok {
		return vm.ToValue(h)
	}
	return
}
