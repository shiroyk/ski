package utils

import (
	"net/http"
	"strings"
)

// ParseCookie parses the given cookie string and return a slice http.Cookie.
func ParseCookie(cookies string) []*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", cookies)
	req := http.Request{Header: header}
	return req.Cookies()
}

// CookieToString returns the serialization string of the slice http.Cookie.
func CookieToString(cookies []*http.Cookie) string {
	switch len(cookies) {
	case 0:
		return ""
	case 1:
		return cookies[0].String()
	}

	var b strings.Builder
	b.WriteString(cookies[0].String())
	for _, cookie := range cookies[1:] {
		b.WriteString(cookie.String())
		b.WriteString("; ")
	}
	return b.String()
}
