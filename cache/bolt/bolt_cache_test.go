package bolt

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_cache")
	assert.NoError(t, os.MkdirAll(tempDir, os.ModePerm))
	defer assert.NoError(t, os.RemoveAll(tempDir))

	c, err := NewCache(cache.Options{Path: tempDir})
	if err != nil {
		t.Fatal(err)
	}

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

	c.SetWithTimeout(key, []byte(value), time.Second)
	v1, _ := c.Get(key)
	assert.Equal(t, value, string(v1))

	time.Sleep(2 * time.Second)

	if _, ok := c.Get(key); ok {
		t.Fatalf("not expired: %v", key)
	}
}
