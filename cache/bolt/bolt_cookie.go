package bolt

import (
	"net/http"
	"net/url"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/lib/logger"
	"github.com/shiroyk/cloudcat/lib/utils"
)

// Cookie is an implementation of cache.Cookie that stores http.Cookie in bolt.DB.
type Cookie struct {
	db *DB
}

// SetCookieString handles the receipt of the cookies string in a reply for the given URL.
func (c *Cookie) SetCookieString(u *url.URL, cookies string) {
	c.SetCookies(u, utils.ParseCookie(cookies))
}

// CookieString returns the cookies string for the given URL.
func (c *Cookie) CookieString(u *url.URL) string {
	value, err := c.db.Get([]byte(u.Host))
	if err != nil {
		return ""
	}
	return string(value)
}

// DeleteCookie delete the cookies for the given URL.
func (c *Cookie) DeleteCookie(u *url.URL) {
	if err := c.db.Delete([]byte(u.Host)); err != nil {
		logger.Errorf("failed to delete cookie %s %s", u.Host, err)
	}
}

// SetCookies handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if err := c.db.Put([]byte(u.Host), []byte(utils.CookieToString(cookies))); err != nil {
		logger.Errorf("failed to set cookie %s %s", u.Host, err)
	}
}

// Cookies returns the cookies to send in a request for the given URL.
func (c *Cookie) Cookies(u *url.URL) []*http.Cookie {
	return utils.ParseCookie(c.CookieString(u))
}

// NewCookie returns a new Cookie that will store cookies in bolt.DB.
func NewCookie(path string) (cache.Cookie, error) {
	db, err := NewDB(path, "cookie", 0)
	if err != nil {
		return nil, err
	}
	return &Cookie{db: db}, nil
}
