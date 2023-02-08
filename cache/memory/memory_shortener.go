package memory

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"sync"
	"time"

	"github.com/shiroyk/cloudcat/logger"
)

// Shortener is an implementation of cache.Shortener that stores HTTP request in in-memory.
type Shortener struct {
	entries *sync.Map
	timeout *sync.Map
}

// Set returns to shorten identifier for the HTTP request.
func (s *Shortener) Set(http string, timeout time.Duration) string {
	hash := md5.Sum([]byte(http)) //nolint:gosec
	id := hex.EncodeToString(hash[:])
	s.entries.Store(id, http)
	s.timeout.Store(id, time.Now().Add(timeout).Unix())
	logger.Debugf("shortener url added %s => %s", id, http)
	return id
}

// Get returns the original HTTP request for the given identifier.
func (s *Shortener) Get(id string) (http string, ok bool) {
	if ddl, exist := s.timeout.Load(id); exist {
		if time.Now().Unix() > ddl.(int64) { //nolint:forcetypeassert
			s.entries.Delete(id)
			s.timeout.Delete(id)
			return
		}
	}
	if h, exist := s.entries.Load(id); exist {
		return h.(string), true //nolint:forcetypeassert
	}
	return
}

// NewShortener returns a new Shortener that will store URL and headers in in-memory.
func NewShortener() *Shortener {
	s := &Shortener{
		entries: new(sync.Map),
		timeout: new(sync.Map),
	}
	return s
}
