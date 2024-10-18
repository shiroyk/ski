package http

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// CookieJar manages storage and use of cookies in HTTP requests.
// Implementations of CookieJar must be safe for concurrent use by multiple
// goroutines.
type CookieJar interface {
	http.CookieJar

	// RemoveCookie delete the cookies for the given URL.
	RemoveCookie(u *url.URL)
}

// memoryCookie is an implementation of CookieJar that stores http.Cookie in in-memory.
type memoryCookie struct {
	*cookiejar.Jar
}

// RemoveCookie remove the cookies for the given URL.
func (c *memoryCookie) RemoveCookie(u *url.URL) {
	exists := c.Cookies(u)
	cookie := make([]*http.Cookie, 0, len(exists))
	for _, e := range exists {
		e.MaxAge = -1
		cookie = append(cookie, e)
	}
	c.SetCookies(u, cookie)
}

// NewCookieJar returns a new CookieJar that will store cookies in in-memory.
func NewCookieJar() CookieJar {
	jar, _ := cookiejar.New(nil)
	return &memoryCookie{jar}
}
