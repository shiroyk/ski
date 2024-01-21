// Package cache the cache JS implementation
package cache

import (
	"errors"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
)

func init() {
	js.Register("cache", &Cache{ski.NewCache()})
}

// Cache interface is used to store string or bytes.
type Cache struct{ ski.Cache }

func (c *Cache) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	if c.Cache == nil {
		return nil, errors.New("Cache can not nil")
	}
	return rt.ToValue(map[string]func(call goja.FunctionCall, vm *goja.Runtime) goja.Value{
		"get":      c.Get,
		"getBytes": c.GetBytes,
		"set":      c.Set,
		"setBytes": c.SetBytes,
		"del":      c.Del,
	}), nil
}

// Get returns string.
func (c *Cache) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if bytes, err := c.Cache.Get(js.Context(vm), call.Argument(0).String()); err == nil && bytes != nil {
		return vm.ToValue(string(bytes))
	}
	return goja.Undefined()
}

// GetBytes returns ArrayBuffer.
func (c *Cache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if bytes, err := c.Cache.Get(js.Context(vm), call.Argument(0).String()); err == nil && bytes != nil {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return goja.Undefined()
}

// Set saves string to the cache with key.
func (c *Cache) Set(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	ctx := js.Context(vm)
	if !goja.IsUndefined(call.Argument(2)) {
		timeout, err := time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
		ctx = ski.WithCacheTimeout(ctx, timeout)
	}

	err := c.Cache.Set(ctx, call.Argument(0).String(), []byte(call.Argument(1).String()))
	if err != nil {
		js.Throw(vm, err)
	}

	return goja.Undefined()
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Cache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	ctx := js.Context(vm)
	if !goja.IsUndefined(call.Argument(2)) {
		timeout, err := time.ParseDuration(call.Argument(2).String())
		if err != nil {
			js.Throw(vm, err)
		}
		ctx = ski.WithCacheTimeout(ctx, timeout)
	}

	value, err := js.ToBytes(call.Argument(1).Export())
	if err != nil {
		js.Throw(vm, err)
	}

	err = c.Cache.Set(ctx, call.Argument(0).String(), value)
	if err != nil {
		js.Throw(vm, err)
	}

	return goja.Undefined()
}

// Del removes key from the cache.
func (c *Cache) Del(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	err := c.Cache.Del(js.Context(vm), call.Argument(0).String())
	if err != nil {
		js.Throw(vm, err)
	}
	return goja.Undefined()
}
