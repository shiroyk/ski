package cloudcat

import (
	"net/http"
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

// ParseCookie parses the cookie string and return a slice http.Cookie.
func ParseCookie(cookies string) []*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", cookies)
	req := http.Request{Header: header}
	return req.Cookies()
}

// ParseSetCookie parses the set-cookie strings and return a slice http.Cookie.
func ParseSetCookie(cookies ...string) []*http.Cookie {
	header := http.Header{}
	for _, cookie := range cookies {
		header.Add("Set-Cookie", cookie)
	}
	res := http.Response{Header: header}
	return res.Cookies()
}

// CookieToString returns the slice string of the slice http.Cookie.
func CookieToString(cookies []*http.Cookie) []string {
	switch len(cookies) {
	case 0:
		return nil
	case 1:
		return []string{cookies[0].String()}
	}

	ret := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		ret = append(ret, cookie.String())
	}
	return ret
}
