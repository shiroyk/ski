package cache

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/shiroyk/cloudcat/core"
	"golang.org/x/exp/slog"
)

// Cookie is an implementation of cache.Cookie that stores http.Cookie in bolt.DB.
type Cookie struct {
	db *DB
}

// CookieString returns the cookies string for the given URL.
func (c *Cookie) CookieString(u *url.URL) []string {
	return cloudcat.CookieToString(c.Cookies(u))
}

// DeleteCookie delete the cookies for the given URL.
func (c *Cookie) DeleteCookie(u *url.URL) {
	if err := c.db.Delete([]byte(u.Host)); err != nil {
		slog.Error(fmt.Sprintf("failed to delete cookie %s %s", u.Host, err))
	}
}

// SetCookies handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if err := c.db.Put([]byte(u.Host), []byte(strings.Join(cloudcat.CookieToString(cookies), "\n"))); err != nil {
		slog.Error(fmt.Sprintf("failed to set cookie %s %s", u.Host, err))
	}
}

// Cookies returns the cookies to send in a request for the given URL.
func (c *Cookie) Cookies(u *url.URL) []*http.Cookie {
	value, err := c.db.Get([]byte(u.Host))
	if err != nil {
		return nil
	}
	return cloudcat.ParseSetCookie(strings.Split(string(value), "\n")...)
}

// NewCookie returns a new Cookie that will store cookies in bolt.DB.
func NewCookie(opt Options) (cloudcat.Cookie, error) {
	db, err := NewDB(opt.Path, "cookie.db", 0)
	if err != nil {
		return nil, err
	}
	return &Cookie{db: db}, nil
}
