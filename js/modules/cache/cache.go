// Package cache the cache JS implementation
package cache

import (
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

// Module js module
type Module struct{}

// Exports returns the module instance
func (*Module) Exports() any {
	return &Cache{cloudcat.MustResolve[cloudcat.Cache]()}
}

func init() {
	jsmodule.Register("cache", &Module{})
}

// Cache interface is used to store string or bytes.
type Cache struct {
	cache cloudcat.Cache
}

// Get returns string.
func (c *Cache) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	opt := cloudcat.CacheOptions{Context: js.VMContext(vm)}
	if bytes, ok := c.cache.Get(call.Argument(0).String(), opt); ok {
		return vm.ToValue(string(bytes))
	}
	return goja.Undefined()
}

// GetBytes returns ArrayBuffer.
func (c *Cache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	opt := cloudcat.CacheOptions{Context: js.VMContext(vm)}
	if bytes, ok := c.cache.Get(call.Argument(0).String(), opt); ok {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return goja.Undefined()
}

// Set saves string to the cache with key.
func (c *Cache) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	var timeout time.Duration
	if !goja.IsUndefined(call.Argument(2)) {
		var err error
		timeout, err = time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
	}

	opt := cloudcat.CacheOptions{Timeout: timeout, Context: js.VMContext(vm)}

	c.cache.Set(call.Argument(0).String(), []byte(call.Argument(1).String()), opt)

	return goja.Undefined()
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Cache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	var timeout time.Duration
	if !goja.IsUndefined(call.Argument(2)) {
		var err error
		timeout, err = time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
	}

	value, err := js.ToBytes(call.Argument(1).Export())
	if err != nil {
		js.Throw(vm, err)
	}

	opt := cloudcat.CacheOptions{Timeout: timeout, Context: js.VMContext(vm)}

	c.cache.Set(call.Argument(0).String(), value, opt)

	return goja.Undefined()
}

// Del removes key from the cache.
func (c *Cache) Del(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	opt := cloudcat.CacheOptions{Context: js.VMContext(vm)}
	c.cache.Del(call.Argument(0).String(), opt)
	return goja.Undefined()
}
