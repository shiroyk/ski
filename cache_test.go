package cloudcat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()
	c := NewCache()

	key, value := "testCacheKey", "testCacheValue"
	if _, ok := c.Get(key); ok {
		t.Fatal("retrieved value before adding it")
	}

	c.Set(key, []byte(value))
	v, _ := c.Get(key)
	assert.Equal(t, value, string(v))

	c.Del(key)
	if _, ok := c.Get(key); ok {
		t.Fatal("delete failed")
	}

	c.Set(key, []byte(value), CacheOptions{Timeout: time.Millisecond})
	v1, _ := c.Get(key)
	assert.Equal(t, value, string(v1))

	time.Sleep(1 * time.Second)

	if _, ok := c.Get(key); ok {
		t.Fatalf("not expired: %v", key)
	}
}
