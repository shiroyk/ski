// Package cache the cache JS implementation
package cache

import (
	"errors"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("cache", &Module{NewCache()})
}

// Module is used to store string or bytes.
type Module struct{ Cache }

func (c *Module) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if c.Cache == nil {
		return nil, errors.New("CacheModule can not nil")
	}
	return rt.ToValue(map[string]func(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value{
		"get":      c.Get,
		"getBytes": c.GetBytes,
		"set":      c.Set,
		"setBytes": c.SetBytes,
		"del":      c.Del,
	}), nil
}

// Get returns string.
func (c *Module) Get(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	if bytes, err := c.Cache.Get(js.Context(vm), call.Argument(0).String()); err == nil && bytes != nil {
		return vm.ToValue(string(bytes))
	}
	return sobek.Undefined()
}

// GetBytes returns ArrayBuffer.
func (c *Module) GetBytes(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	if bytes, err := c.Cache.Get(js.Context(vm), call.Argument(0).String()); err == nil && bytes != nil {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return sobek.Undefined()
}

// Set saves string to the cache with key.
func (c *Module) Set(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	ctx := js.Context(vm)

	var timeout time.Duration
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		timeout = time.Millisecond * time.Duration(v.ToInteger())
	}

	err := c.Cache.Set(ctx, call.Argument(0).String(), []byte(call.Argument(1).String()), timeout)
	if err != nil {
		js.Throw(vm, err)
	}

	return sobek.Undefined()
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Module) SetBytes(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	ctx := js.Context(vm)
	value, err := js.ToBytes(call.Argument(1).Export())
	if err != nil {
		js.Throw(vm, err)
	}

	var timeout time.Duration
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		timeout = time.Millisecond * time.Duration(v.ToInteger())
	}

	err = c.Cache.Set(ctx, call.Argument(0).String(), value, timeout)
	if err != nil {
		js.Throw(vm, err)
	}

	return sobek.Undefined()
}

// Del removes key from the cache.
func (c *Module) Del(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	err := c.Cache.Del(js.Context(vm), call.Argument(0).String())
	if err != nil {
		js.Throw(vm, err)
	}
	return sobek.Undefined()
}
