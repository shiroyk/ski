package ski

import (
	"context"
	"sync"
	"time"
)

// A Cache interface is used to store bytes.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, timeout time.Duration) error
	Del(ctx context.Context, key string) error
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
		if time.Now().UnixMilli() > ddl {
			delete(c.items, key)
			delete(c.timeout, key)
			return nil, nil
		}
	}
	return c.items[key], nil
}

// Set saves []byte to the cache with key
func (c *memoryCache) Set(_ context.Context, key string, value []byte, timeout time.Duration) error {
	c.Lock()
	defer c.Unlock()
	c.items[key] = value
	c.timeout[key] = time.Now().Add(timeout).UnixMilli()
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
