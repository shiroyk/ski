package core

import (
	"net/http"
	"strings"
)

// ZeroOr if value is zero value returns the defaultValue
func ZeroOr[T comparable](value, defaultValue T) T {
	var zero T
	if zero == value {
		return defaultValue
	}
	return value
}

// EmptyOr if slice is empty returns the defaultValue
func EmptyOr[T any](value, defaultValue []T) []T {
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

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
		b.WriteString("; ")
		b.WriteString(cookie.String())
	}
	return b.String()
}
