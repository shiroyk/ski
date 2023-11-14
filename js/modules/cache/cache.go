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
	if bytes, ok := c.cache.Get(js.VMContext(vm), call.Argument(0).String()); ok {
		return vm.ToValue(string(bytes))
	}
	return goja.Undefined()
}

// GetBytes returns ArrayBuffer.
func (c *Cache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if bytes, ok := c.cache.Get(js.VMContext(vm), call.Argument(0).String()); ok {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return goja.Undefined()
}

// Set saves string to the cache with key.
func (c *Cache) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	ctx := js.VMContext(vm)
	if !goja.IsUndefined(call.Argument(2)) {
		timeout, err := time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
		ctx = cloudcat.WithCacheTimeout(ctx, timeout)
	}

	c.cache.Set(ctx, call.Argument(0).String(), []byte(call.Argument(1).String()))

	return goja.Undefined()
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Cache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	ctx := js.VMContext(vm)
	if !goja.IsUndefined(call.Argument(2)) {
		timeout, err := time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
		ctx = cloudcat.WithCacheTimeout(ctx, timeout)
	}

	value, err := js.ToBytes(call.Argument(1).Export())
	if err != nil {
		js.Throw(vm, err)
	}

	c.cache.Set(ctx, call.Argument(0).String(), value)

	return goja.Undefined()
}

// Del removes key from the cache.
func (c *Cache) Del(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	c.cache.Del(js.VMContext(vm), call.Argument(0).String())
	return goja.Undefined()
}
