package bolt

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/lib/logger"
)

// Shortener is an implementation of cache.Shortener that stores HTTP request in bolt.DB.
type Shortener struct {
	db *DB
}

// Set returns to shorten identifier for the given HTTP request with timeout (unit second).
func (s *Shortener) Set(http string, timeout time.Duration) string {
	hash := md5.Sum([]byte(http)) //nolint:gosec
	id := hex.EncodeToString(hash[:])

	if err := s.db.PutWithTimeout([]byte(id), []byte(http), timeout); err != nil {
		logger.Errorf("failed to set shortener %s", err)
	}
	return id
}

// Get returns the original HTTP request for the given identifier.
func (s *Shortener) Get(id string) (http string, ok bool) {
	value, err := s.db.Get([]byte(id))
	if err != nil {
		return "", false
	}
	return string(value), true
}

// NewShortener returns a new Shortener that will store URL and headers in bolt.DB.
func NewShortener(opt cache.Options) (cache.Shortener, error) {
	db, err := NewDB(opt.Path, "shortener", defaultInterval)
	if err != nil {
		return nil, err
	}
	return &Shortener{db: db}, nil
}
