package ski

import (
	"context"
	"fmt"
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

// RegisterCache registers cache.get and cache.set with the Cache.
func RegisterCache(cache Cache) {
	Registers(NewExecutors{
		"cache.get": cache_get(cache),
		"cache.set": cache_set(cache),
	})
}

type _cache_get struct {
	Cache
	key string
}

// cache_get returns the string from the cache with key
func cache_get(cache Cache) NewExecutor {
	return func(args Arguments) (Executor, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cache.get: invalid arguments")
		}
		return _cache_get{cache, args.GetString(0)}, nil
	}
}

func (c _cache_get) Exec(ctx context.Context, _ any) (any, error) {
	data, err := c.Get(ctx, c.key)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

type _cache_set struct {
	Cache
	key     string
	timeout time.Duration
}

// cache_set saves string to the cache with key
func cache_set(cache Cache) NewExecutor {
	return func(args Arguments) (Executor, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("cache.set: invalid arguments")
		}
		var timeout time.Duration
		if len(args) > 1 {
			var err error
			timeout, err = cast.ToDurationE(args.GetString(1))
			if err != nil {
				return nil, err
			}
		}
		return _cache_set{cache, args.GetString(0), timeout}, nil
	}
}

func (c _cache_set) Exec(ctx context.Context, arg any) (any, error) {
	var data []byte
	switch t := arg.(type) {
	case string:
		data = []byte(t)
	case []byte:
		data = t
	case fmt.Stringer:
		data = []byte(t.String())
	default:
		return nil, fmt.Errorf("cache.set: invalid type %T", arg)
	}

	if c.timeout > 0 {
		ctx = WithCacheTimeout(ctx, c.timeout)
	}

	return nil, c.Set(ctx, c.key, data)
}
