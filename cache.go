package cloudcat

import (
	"context"
	"sync"
	"time"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cast"
)

// A Cache interface is used to store bytes.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte)
	Del(ctx context.Context, key string)
}

var cacheTimeoutKey struct{}

// WithCacheTimeout returns the context with the cache timeout.
func WithCacheTimeout(ctx context.Context, timeout time.Duration) context.Context {
	if c, ok := ctx.(*plugin.Context); ok {
		c.SetValue(&cacheTimeoutKey, timeout)
		return ctx
	}
	return context.WithValue(ctx, &cacheTimeoutKey, timeout)
}

// CacheTimeout returns the context cache timeout value.
func CacheTimeout(ctx context.Context) time.Duration {
	return cast.ToDuration(ctx.Value(&cacheTimeoutKey))
}

// memoryCache is an implementation of Cache that stores bytes in in-memory.
type memoryCache struct {
	sync.Mutex
	items   map[string][]byte
	timeout map[string]int64
}

// Get returns the []byte and true, if not existing returns false.
func (c *memoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	c.Lock()
	defer c.Unlock()
	if ddl, exist := c.timeout[key]; exist {
		if time.Now().Unix() > ddl {
			delete(c.items, key)
			delete(c.timeout, key)
			return []byte{}, false
		}
	}
	if b, ok := c.items[key]; ok {
		return b, true
	}
	return []byte{}, false
}

// Set saves []byte to the cache with key
func (c *memoryCache) Set(ctx context.Context, key string, value []byte) {
	c.Lock()
	c.items[key] = value
	if timeout := CacheTimeout(ctx); timeout > 0 {
		c.timeout[key] = time.Now().Add(timeout).Unix()
	}
	c.Unlock()
}

// Del removes key from the cache
func (c *memoryCache) Del(_ context.Context, key string) {
	c.Lock()
	delete(c.items, key)
	delete(c.timeout, key)
	c.Unlock()
}

// NewCache returns a new Cache that will store items in in-memory.
func NewCache() Cache {
	return &memoryCache{
		items:   make(map[string][]byte),
		timeout: make(map[string]int64),
	}
}
