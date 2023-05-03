package core

import (
	"sync"
	"time"
)

// A Cache interface is used to store bytes.
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	SetWithTimeout(key string, value []byte, timeout time.Duration)
	Del(key string)
}

// memoryCache is an implementation of Cache that stores bytes in in-memory.
type memoryCache struct {
	sync.Mutex
	items   map[string][]byte
	timeout map[string]int64
}

// Get returns the []byte and true, if not existing returns false.
func (c *memoryCache) Get(key string) ([]byte, bool) {
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
func (c *memoryCache) Set(key string, value []byte) {
	c.Lock()
	c.items[key] = value
	c.Unlock()
}

// SetWithTimeout saves []byte to the cache with key
func (c *memoryCache) SetWithTimeout(key string, value []byte, timeout time.Duration) {
	c.Lock()
	c.items[key] = value
	c.timeout[key] = time.Now().Add(timeout).Unix()
	c.Unlock()
}

// Del removes key from the cache
func (c *memoryCache) Del(key string) {
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
