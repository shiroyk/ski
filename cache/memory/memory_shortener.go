package memory

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/labstack/gommon/log"
	"github.com/shiroyk/cloudcat/utils"
	"golang.org/x/exp/slog"
)

// Shortener is an implementation of cache.Shortener that stores URL and headers in in-memory.
type Shortener struct {
	lruCache *utils.LRUCache[string, entry]
}

// entry struct URL and headers.
type entry struct {
	Url     string
	Headers map[string]string
}

// Set returns to shorten identifier for the given URL and headers.
func (shortener *Shortener) Set(url string, headers map[string]string) string {
	hash := md5.Sum([]byte(url))
	id := hex.EncodeToString(hash[:])
	shortener.lruCache.Add(id, entry{Url: url, Headers: headers})
	log.Debugf("shortener url added %s => %s", id, url)
	return id
}

// Get returns the original URL and headers for the given identifier.
func (shortener *Shortener) Get(id string) (url string, headers map[string]string, ok bool) {
	if e, ok := shortener.lruCache.Get(id); ok {
		url = e.Url
		headers = e.Headers
		return url, headers, true
	}
	return
}

// NewShortener returns a new Shortener that will store URL and headers in in-memory.
func NewShortener() *Shortener {
	s := &Shortener{
		lruCache: utils.NewLRUCache[string, entry](128),
	}
	s.lruCache.OnEvicted = func(key string, value entry) {
		slog.Debug("shortener clean key %s", key)
	}
	return s
}
