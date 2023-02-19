package cache

import (
	"fmt"

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

// Global returns is it is a global module
func (*Module) Global() bool {
	return false
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
func (c *Cache) Set(call goja.FunctionCall) (ret goja.Value) {
	c.cache.Set(call.Argument(0).String(), []byte(call.Argument(1).String()))
	return
}

// SetBytes saves ArrayBuffer to the cache with key.
func (c *Cache) SetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if buffer, ok := call.Argument(1).Export().(goja.ArrayBuffer); ok {
		c.cache.Set(call.Argument(0).String(), buffer.Bytes())
	} else {
		common.Throw(vm, fmt.Errorf("setBytes unsupport type %T", call.Argument(1).Export()))
	}
	return
}

// Del removes key from the cache.
func (c *Cache) Del(call goja.FunctionCall) (ret goja.Value) {
	c.cache.Del(call.Argument(0).String())
	return
}
