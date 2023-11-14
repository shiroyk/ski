package cloudcat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()
	c := NewCache()
	ctx := context.Background()

	key, value := "testCacheKey", "testCacheValue"
	if _, ok := c.Get(ctx, key); ok {
		t.Fatal("retrieved value before adding it")
	}

	c.Set(ctx, key, []byte(value))
	v, _ := c.Get(ctx, key)
	assert.Equal(t, value, string(v))

	c.Del(ctx, key)
	if _, ok := c.Get(ctx, key); ok {
		t.Fatal("delete failed")
	}

	c.Set(WithCacheTimeout(ctx, time.Millisecond), key, []byte(value))
	v1, _ := c.Get(ctx, key)
	assert.Equal(t, value, string(v1))

	time.Sleep(1 * time.Second)

	if _, ok := c.Get(ctx, key); ok {
		t.Fatalf("not expired: %v", key)
	}
}
