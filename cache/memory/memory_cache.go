package memory

import (
	"sync"
)

// Cache is an implementation of Cache that stores bytes in in-memory.
type Cache struct {
	sync.RWMutex
	items map[string][]byte
}

// Get returns the []byte and true, if not existing returns false.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.RLock()
	defer c.RUnlock()
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

// Del removes key from the cache
func (c *Cache) Del(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}

// NewCache returns a new Cache that will store items in in-memory.
func NewCache() *Cache {
	return &Cache{
		items: make(map[string][]byte),
	}
}
