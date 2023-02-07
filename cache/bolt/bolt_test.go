package bolt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	t.Parallel()
	temp := filepath.Join(os.TempDir(), "bolt")
	os.MkdirAll(temp, os.ModePerm)
	defer os.RemoveAll(temp)

	db, err := NewDB(temp, "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("testKey")
	value := []byte("testValue")

	err = db.PutWithTimeout(key, value, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	getValue, err := db.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value, getValue) {
		t.Fatalf("want %v, got %v", string(getValue), string(value))
	}

	time.Sleep(2 * time.Second)

	_, err = db.Get(key)
	if err != ErrKeyNotFound {
		t.Fatalf("key not expired, got error %v", err)
	}
}
