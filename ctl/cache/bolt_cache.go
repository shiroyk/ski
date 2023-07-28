package cache

import (
	"fmt"
	"time"

	"github.com/shiroyk/cloudcat/core"
	"golang.org/x/exp/slog"
)

// Cache is an implementation of Cache that stores bytes in bolt.DB.
type Cache struct {
	db *DB
}

// Get returns the []byte and true, if not existing returns false.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	var err error
	if value, err = c.db.Get([]byte(key)); err != nil || value == nil {
		return []byte{}, false
	}
	return value, true
}

// Set saves []byte to the cache with key.
func (c *Cache) Set(key string, value []byte, opts ...cloudcat.CacheOptions) {
	var timeout time.Duration
	if len(opts) > 0 && opts[0].Timeout > 0 {
		timeout = opts[0].Timeout
	}
	err := c.db.PutWithTimeout([]byte(key), value, timeout)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to set cache with key %s %s", key, err))
	}
}

// Del removes key from the cache.
func (c *Cache) Del(key string) {
	err := c.db.Delete([]byte(key))
	if err != nil {
		slog.Error(fmt.Sprintf("failed to delete cache with key %s %s", key, err))
	}
}

// NewCache returns a new Cache that will store items in bolt.DB.
func NewCache(opt Options) (cloudcat.Cache, error) {
	db, err := NewDB(opt.Path, "cache.db", cloudcat.ZeroOr(opt.ExpireCleanInterval, defaultInterval))
	if err != nil {
		return nil, err
	}
	return &Cache{db: db}, nil
}
