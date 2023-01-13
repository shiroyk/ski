package utils

/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"container/list"
	"sync"
)

// LRUCache is an LRU cache. It is safe for concurrent access.
type LRUCache[K comparable, V any] struct {
	sync.RWMutex
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key K, value V)

	ll    *list.List
	cache map[K]*list.Element
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUCache creates a new LRUCache.
// If maxEntries is zero, the cache has no limit, and it's assumed
// that eviction is done by the caller.
func NewLRUCache[K comparable, V any](maxEntries int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[K]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *LRUCache[K, V]) Add(key K, value V) {
	c.Lock()
	defer c.Unlock()
	if c.cache == nil {
		c.cache = make(map[K]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry[K, V]).value = value
		return
	}
	ele := c.ll.PushFront(&entry[K, V]{key, value})
	c.cache[key] = ele
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.cache == nil {
		return
	}
	c.RLock()
	defer c.RUnlock()
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry[K, V]).value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *LRUCache[K, V]) Remove(key K) {
	if c.cache == nil {
		return
	}
	c.Lock()
	defer c.Unlock()
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRUCache[K, V]) RemoveOldest() {
	if c.cache == nil {
		return
	}
	c.Lock()
	defer c.Unlock()
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *LRUCache[K, V]) removeElement(e *list.Element) {
	c.Lock()
	defer c.Unlock()
	c.ll.Remove(e)
	kv := e.Value.(*entry[K, V])
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *LRUCache[K, V]) Len() int {
	if c.cache == nil {
		return 0
	}
	c.RLock()
	defer c.RUnlock()
	return c.ll.Len()
}

// Clear purges all stored items from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.Lock()
	defer c.Unlock()
	if c.OnEvicted != nil {
		for _, e := range c.cache {
			kv := e.Value.(*entry[K, V])
			c.OnEvicted(kv.key, kv.value)
		}
	}
	c.ll = nil
	c.cache = nil
}
