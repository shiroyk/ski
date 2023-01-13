package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
)

type jsCache struct {
	cache cache.Cache
}

// Get returns string.
func (c *jsCache) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if bytes, ok := c.cache.Get(call.Argument(0).String()); ok {
		return vm.ToValue(string(bytes))
	}
	return
}

// GetBytes returns ArrayBuffer.
func (c *jsCache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if bytes, ok := c.cache.Get(call.Argument(0).String()); ok {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return
}

// Set saves string to the cache with key.
func (c *jsCache) Set(call goja.FunctionCall) (ret goja.Value) {
	c.cache.Set(call.Argument(0).String(), []byte(call.Argument(1).String()))
	return
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *jsCache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if buffer, ok := call.Argument(1).Export().(goja.ArrayBuffer); ok {
		c.cache.Set(call.Argument(0).String(), buffer.Bytes())
	} else {
		panic(vm.ToValue(fmt.Errorf("setBytes unsupport type %T", call.Argument(1).Export())))
	}
	return
}

// Del removes key from the cache.
func (c *jsCache) Del(call goja.FunctionCall) (ret goja.Value) {
	c.cache.Del(call.Argument(0).String())
	return
}
