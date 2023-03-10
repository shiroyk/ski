package bolt

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	t.Parallel()
	temp := filepath.Join(os.TempDir(), "bolt")
	assert.NoError(t, os.MkdirAll(temp, os.ModePerm))
	defer assert.NoError(t, os.RemoveAll(temp))

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
	assert.Equal(t, getValue, value)

	time.Sleep(2 * time.Second)

	_, err = db.Get(key)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}
