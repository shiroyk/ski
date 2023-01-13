package leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
)

// Cache is an implementation of Cache that stores bytes in leveldb.DB.
type Cache struct {
	Db *leveldb.DB
}

// Get returns the []byte and true, if not existing returns false.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	var err error
	value, err = c.Db.Get([]byte(key), nil)
	if err != nil {
		return []byte{}, false
	}
	return value, true
}

// Set saves []byte to the cache with key.
func (c *Cache) Set(key string, value []byte) {
	_ = c.Db.Put([]byte(key), value, nil)
}

// Del removes key from the cache.
func (c *Cache) Del(key string) {
	_ = c.Db.Delete([]byte(key), nil)
}

// NewCache returns a new Cache that will store items in leveldb.DB.
func NewCache(path string) (*Cache, error) {
	c := &Cache{}

	var err error
	c.Db, err = leveldb.OpenFile(path, nil)

	if err != nil {
		return nil, err
	}
	return c, nil
}
