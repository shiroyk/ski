package cache

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
)

// Module js module
type Module struct{}

// Exports returns the module instance
func (*Module) Exports() any {
	return &Cache{di.MustResolve[cache.Cache]()}
}

func init() {
	modules.Register("cache", &Module{})
}

// Cache interface is used to store string or bytes.
type Cache struct {
	cache cache.Cache
}

// Get returns string.
func (c *Cache) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if bytes, ok := c.cache.Get(call.Argument(0).String()); ok {
		return vm.ToValue(string(bytes))
	}
	return
}

// GetBytes returns ArrayBuffer.
func (c *Cache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if bytes, ok := c.cache.Get(call.Argument(0).String()); ok {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return
}

// Set saves string to the cache with key.
func (c *Cache) Set(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	value := call.Argument(1).String()
	ddl := call.Argument(2)

	var timeout time.Duration
	if !goja.IsUndefined(ddl) {
		var err error
		timeout, err = time.ParseDuration(ddl.String())
		if err != nil {
			common.Throw(vm, err)
		}
	}

	if timeout > 0 {
		c.cache.SetWithTimeout(key, []byte(value), timeout)
		return
	}

	c.cache.Set(key, []byte(value))
	return
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Cache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	buffer, ok := call.Argument(1).Export().(goja.ArrayBuffer)
	ddl := call.Argument(2)

	if !ok {
		common.Throw(vm, fmt.Errorf("setBytes unsupport type %T", call.Argument(1).Export()))
	}

	var timeout time.Duration
	if !goja.IsUndefined(ddl) {
		var err error
		timeout, err = time.ParseDuration(ddl.String())
		if err != nil {
			common.Throw(vm, err)
		}
	}

	if timeout > 0 {
		c.cache.SetWithTimeout(key, buffer.Bytes(), timeout)
		return
	}

	c.cache.Set(call.Argument(0).String(), buffer.Bytes())
	return
}

// Del removes key from the cache.
func (c *Cache) Del(call goja.FunctionCall) (ret goja.Value) {
	c.cache.Del(call.Argument(0).String())
	return
}
