package js

import (
	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/spf13/cast"
)

type jsShortener struct {
	shortener cache.Shortener
}

func (s *jsShortener) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	header := cast.ToStringMapString(call.Argument(1).Export())
	return vm.ToValue(s.shortener.Set(call.Argument(0).String(), header))
}

func (s *jsShortener) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if url, headers, ok := s.shortener.Get(call.Argument(0).String()); ok {
		return vm.ToValue(map[string]any{
			"url":     url,
			"headers": headers,
		})
	}
	return vm.ToValue(map[string]any{})
}
