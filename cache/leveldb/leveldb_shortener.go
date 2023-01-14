package leveldb

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/syndtr/goleveldb/leveldb"
)

// Shortener is an implementation of cache.Shortener that stores HTTP request in leveldb.DB.
type Shortener struct {
	Db *leveldb.DB
}

// Set returns to shorten identifier for the given HTTP  request.
func (s *Shortener) Set(http string) string {
	hash := md5.Sum([]byte(http))
	id := hex.EncodeToString(hash[:])
	_ = s.Db.Put([]byte(id), []byte(http), nil)
	return id
}

// Get returns the original HTTP request for the given identifier.
func (s *Shortener) Get(id string) (http string, ok bool) {
	value, err := s.Db.Get([]byte(id), nil)
	if err != nil {
		return "", false
	}
	return string(value), true
}

// NewShortener returns a new Shortener that will store URL and headers in leveldb.DB.
func NewShortener(path string) (*Shortener, error) {
	shortener := &Shortener{}

	var err error
	shortener.Db, err = leveldb.OpenFile(path, nil)

	if err != nil {
		return nil, err
	}
	return shortener, nil
}
