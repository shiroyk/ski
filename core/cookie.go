package core

import (
	"net/http"
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
