package leveldb

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/shiroyk/cloudcat/utils"
	"github.com/syndtr/goleveldb/leveldb"
)

// Cookie is an implementation of cache.Cookie that stores http.Cookie in leveldb.DB.
type Cookie struct {
	mu sync.RWMutex
	Db *leveldb.DB
}

// SetCookieString handles the receipt of the cookies strung in a reply for the given URL.
func (c *Cookie) SetCookieString(u *url.URL, cookies string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SetCookies(u, utils.ParseCookie(cookies))
}

// CookieString returns the cookies string to send in a request for the given URL.
func (c *Cookie) CookieString(u *url.URL) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, err := c.Db.Get([]byte(u.Host), nil)
	if err != nil {
		return ""
	}
	return string(value)
}

// DeleteCookie handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) DeleteCookie(u *url.URL) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.Db.Delete([]byte(u.Host), nil)
}

// SetCookies handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.Db.Put([]byte(u.Host), []byte(utils.CookieToString(cookies)), nil)
}

// Cookies returns the cookies to send in a request for the given URL.
func (c *Cookie) Cookies(u *url.URL) []*http.Cookie {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return utils.ParseCookie(c.CookieString(u))
}

// NewCookie returns a new Cookie that will store cookies in leveldb.DB.
func NewCookie(path string) (*Cookie, error) {
	cookie := &Cookie{}

	var err error
	cookie.Db, err = leveldb.OpenFile(path, nil)

	if err != nil {
		return nil, err
	}
	return cookie, nil
}
