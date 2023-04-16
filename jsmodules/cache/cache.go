// Package cache the cache JS implementation
package cache

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

// Module js module
type Module struct{}

// Exports returns the module instance
func (*Module) Exports() any {
	return &Cache{core.MustResolve[core.Cache]()}
}

func init() {
	jsmodule.Register("cache", &Module{})
}

// Cache interface is used to store string or bytes.
type Cache struct {
	cache core.Cache
}

// Get returns string.
func (c *Cache) Get(name string) string {
	if bytes, ok := c.cache.Get(name); ok {
		return string(bytes)
	}
	return ""
}

// GetBytes returns ArrayBuffer.
func (c *Cache) GetBytes(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if bytes, ok := c.cache.Get(call.Argument(0).String()); ok {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}
	return
}

// Set saves string to the cache with key.
func (c *Cache) Set(key, value, ddl string) (err error) {
	var timeout time.Duration
	if ddl != "" {
		timeout, err = time.ParseDuration(ddl)
		if err != nil {
			return
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
func (c *Cache) SetBytes(key string, value any, ddl string) (err error) {
	var timeout time.Duration
	buffer, ok := value.(goja.ArrayBuffer)
	if !ok {
		return fmt.Errorf("setBytes value type unsupport")
	}
	if ddl != "" {
		timeout, err = time.ParseDuration(ddl)
		if err != nil {
			return
		}
	}

	if timeout > 0 {
		c.cache.SetWithTimeout(key, buffer.Bytes(), timeout)
		return
	}

	c.cache.Set(key, buffer.Bytes())
	return
}

// Del removes key from the cache.
func (c *Cache) Del(key string) {
	c.cache.Del(key)
}
