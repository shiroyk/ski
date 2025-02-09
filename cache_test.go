package ski

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
	if v, _ := c.Get(ctx, key); len(v) != 0 {
		t.Fatal("retrieved values before adding it")
	}

	_ = c.Set(ctx, key, []byte(value), 0)
	v, _ := c.Get(ctx, key)
	assert.Equal(t, value, string(v))

	_ = c.Del(ctx, key)
	v, _ = c.Get(ctx, key)
	assert.Empty(t, v)

	_ = c.Set(ctx, key, []byte(value), time.Millisecond)
	v1, _ := c.Get(ctx, key)
	assert.Equal(t, value, string(v1))

	time.Sleep(100 * time.Millisecond)

	v, _ = c.Get(ctx, key)
	assert.Empty(t, v, "not expired: %v", key)
}
