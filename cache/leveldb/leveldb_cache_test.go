package leveldb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "cache")
	defer os.RemoveAll(tempDir)

	c, err := NewCache(filepath.Join(tempDir, "Db"))
	if err != nil {
		t.Fatalf("New leveldb: %v", err)
	}

	key, value := "testKey", "testValue"
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
