package cloudcat

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	*cookiejar.Jar
}

// SetCookieString handles the receipt of the cookies string in a reply for the given URL.
func (c *memoryCookie) SetCookieString(u *url.URL, cookies string) {
	c.SetCookies(u, ParseCookie(cookies))
}

// CookieString returns the cookies string for the given URL.
func (c *memoryCookie) CookieString(u *url.URL) string {
	return CookieToString(c.Cookies(u))
}

// DeleteCookie delete the cookies for the given URL.
func (c *memoryCookie) DeleteCookie(u *url.URL) {
	exists := c.Cookies(u)
	cookie := make([]*http.Cookie, 0, len(exists))
	for _, e := range exists {
		e.MaxAge = -1
		cookie = append(cookie, e)
	}
	c.SetCookies(u, cookie)
}

// NewCookie returns a new Cookie that will store cookies in in-memory.
func NewCookie() Cookie {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &memoryCookie{jar}
}
