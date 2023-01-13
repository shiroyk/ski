package leveldb

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	
	"github.com/syndtr/goleveldb/leveldb"
)

// Shortener is an implementation of cache.Shortener that stores URL and headers in leveldb.DB.
type Shortener struct {
	Db *leveldb.DB
}

// entry struct URL and headers.
type entry struct {
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Set returns to shorten identifier for the given URL and headers.
func (s *Shortener) Set(url string, headers map[string]string) string {
	hash := md5.Sum([]byte(url))
	id := hex.EncodeToString(hash[:])
	value, _ := json.Marshal(entry{Url: url, Headers: headers})
	_ = s.Db.Put([]byte(id), value, nil)
	return id
}

// Get returns the original URL and headers for the given identifier.
func (s *Shortener) Get(id string) (url string, headers map[string]string, ok bool) {
	value, err := s.Db.Get([]byte(id), nil)
	if err != nil {
		return "", nil, false
	}
	var e entry
	_ = json.Unmarshal(value, &e)
	return e.Url, e.Headers, true
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
