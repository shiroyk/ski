package core

import (
	"net/http"
	"net/url"
	"sync"
)

// Cookie manages storage and use of cookies in HTTP requests.
// Implementations of Cookie must be safe for concurrent use by multiple
// goroutines.
type Cookie interface {
	http.CookieJar

	// SetCookieString handles the receipt of the cookies string in a reply for the given URL.
	SetCookieString(u *url.URL, cookies string)
	// CookieString returns the cookies string for the given URL.
	CookieString(u *url.URL) string
	// DeleteCookie delete the cookies for the given URL.
	DeleteCookie(u *url.URL)
}

// memoryCookie is an implementation of Cookie that stores http.Cookie in in-memory.
type memoryCookie struct {
	entries *sync.Map
}

// SetCookieString handles the receipt of the cookies string in a reply for the given URL.
func (c *memoryCookie) SetCookieString(u *url.URL, cookies string) {
	c.entries.Store(u.Host, cookies)
}

// CookieString returns the cookies string for the given URL.
func (c *memoryCookie) CookieString(u *url.URL) string {
	if cookies, ok := c.entries.Load(u.Host); ok {
		return cookies.(string)
	}
	return ""
}

// DeleteCookie delete the cookies for the given URL.
func (c *memoryCookie) DeleteCookie(u *url.URL) {
	c.entries.Delete(u.Host)
}

// SetCookies handles the receipt of the cookies in a reply for the given URL.
func (c *memoryCookie) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.entries.Store(u.Host, CookieToString(cookies))
}

// Cookies returns the cookies to send in a request for the given URL.
func (c *memoryCookie) Cookies(u *url.URL) []*http.Cookie {
	if cookies, ok := c.entries.Load(u.Host); ok {
		return ParseCookie(cookies.(string))
	}
	return nil
}

// NewCookie returns a new Cookie that will store cookies in in-memory.
func NewCookie() Cookie {
	return &memoryCookie{entries: new(sync.Map)}
}
