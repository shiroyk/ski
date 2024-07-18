package ski

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/cast"
)

// A Cache interface is used to store bytes.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Del(ctx context.Context, key string) error
}

var cacheTimeoutKey byte

// WithCacheTimeout returns the context with the cache timeout.
func WithCacheTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return WithValue(ctx, &cacheTimeoutKey, timeout)
}

// CacheTimeout returns the context cache timeout values.
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
func (c *memoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.Lock()
	defer c.Unlock()
	if ddl, exist := c.timeout[key]; exist {
		if time.Now().Unix() > ddl {
			delete(c.items, key)
			delete(c.timeout, key)
			return []byte{}, nil
		}
	}
	if b, ok := c.items[key]; ok {
		return b, nil
	}
	return nil, nil
}

// Set saves []byte to the cache with key
func (c *memoryCache) Set(ctx context.Context, key string, value []byte) error {
	c.Lock()
	defer c.Unlock()
	c.items[key] = value
	if timeout := CacheTimeout(ctx); timeout > 0 {
		c.timeout[key] = time.Now().Add(timeout).Unix()
	}
	return nil
}

// Del removes key from the cache
func (c *memoryCache) Del(_ context.Context, key string) error {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
	delete(c.timeout, key)
	return nil
}

// NewCache returns a new Cache that will store items in in-memory.
func NewCache() Cache {
	return &memoryCache{
		items:   make(map[string][]byte),
		timeout: make(map[string]int64),
	}
}
