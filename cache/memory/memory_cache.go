// Package memory the memory key/value store
package memory

import (
	"sync"
	"time"
)

// Cache is an implementation of Cache that stores bytes in in-memory.
type Cache struct {
	sync.RWMutex
	items   map[string][]byte
	timeout map[string]int64
}

// Get returns the []byte and true, if not existing returns false.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.RLock()
	defer c.RUnlock()
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
func (c *Cache) Set(key string, value []byte) {
	c.Lock()
	c.items[key] = value
	c.Unlock()
}

// SetWithTimeout saves []byte to the cache with key
func (c *Cache) SetWithTimeout(key string, value []byte, timeout time.Duration) {
	c.Lock()
	c.items[key] = value
	c.timeout[key] = time.Now().Add(timeout).Unix()
	c.Unlock()
}

// Del removes key from the cache
func (c *Cache) Del(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}

// NewCache returns a new Cache that will store items in in-memory.
func NewCache() *Cache {
	return &Cache{
		items:   make(map[string][]byte),
		timeout: make(map[string]int64),
	}
}
