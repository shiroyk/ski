package memory

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/shiroyk/cloudcat/utils"
	"golang.org/x/exp/slog"
)

// Shortener is an implementation of cache.Shortener that stores HTTP request in in-memory.
type Shortener struct {
	lruCache *utils.LRUCache[string, string]
}

// Set returns to shorten identifier for the HTTP request.
func (shortener *Shortener) Set(http string) string {
	hash := md5.Sum([]byte(http))
	id := hex.EncodeToString(hash[:])
	shortener.lruCache.Add(id, http)
	slog.Debug("shortener url added %s => %s", id, http)
	return id
}

// Get returns the original HTTP request for the given identifier.
func (shortener *Shortener) Get(id string) (http string, ok bool) {
	if h, ok := shortener.lruCache.Get(id); ok {
		return h, true
	}
	return
}

// NewShortener returns a new Shortener that will store URL and headers in in-memory.
func NewShortener() *Shortener {
	s := &Shortener{
		lruCache: utils.NewLRUCache[string, string](128),
	}
	s.lruCache.OnEvicted = func(key string, value string) {
		slog.Debug("shortener clean key %s", key)
	}
	return s
}
