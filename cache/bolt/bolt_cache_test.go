package bolt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_cache")
	os.MkdirAll(tempDir, os.ModePerm)
	defer os.RemoveAll(tempDir)

	c, err := NewCache(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	key, value := "testCacheKey", "testCacheValue"
	if _, ok := c.Get(key); ok {
		t.Fatal("retrieved value before adding it")
	}

	c.Set(key, []byte(value))
	v, _ := c.Get(key)
	if string(v) != value {
		t.Fatalf("unexpected value %s", v)
	}

	c.Del(key)
	if _, ok := c.Get(key); ok {
		t.Fatal("delete failed")
	}
}
