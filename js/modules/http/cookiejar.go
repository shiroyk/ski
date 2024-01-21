package http

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/spf13/cast"
)

// CookieJar manages storage and use of cookies in HTTP requests.
type CookieJar struct{ ski.CookieJar }

func (j *CookieJar) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	if j.CookieJar == nil {
		return nil, errors.New("CookieJar can not nil")
	}
	return rt.ToValue(map[string]func(call goja.FunctionCall, rt *goja.Runtime) goja.Value{
		"get":    j.Get,
		"getAll": j.GetAll,
		"set":    j.Set,
		"del":    j.Del,
	}), nil
}

// Get returns the cookie for the given option.
func (j *CookieJar) Get(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	opt, err := cast.ToStringMapStringE(call.Argument(0).Export())
	if err != nil {
		js.Throw(rt, errors.New("get parameter must be an object containing name, url"))
	}
	u, err := url.Parse(opt["url"])
	if err != nil {
		js.Throw(rt, err)
	}
	cookies := j.CookieJar.Cookies(u)
	name := opt["name"]
	for _, cookie := range cookies {
		if cookie.Name == name {
			return toObj(cookie, rt)
		}
	}
	if len(cookies) > 0 {
		return toObj(cookies[0], rt)
	}
	return goja.Null()
}

// GetAll returns the cookies for the given option.
func (j *CookieJar) GetAll(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	opt, err := cast.ToStringMapStringE(call.Argument(0).Export())
	if err != nil {
		js.Throw(rt, errors.New("getAll parameter must be an object containing name, url"))
	}
	u, err := url.Parse(opt["url"])
	if err != nil {
		js.Throw(rt, err)
	}
	return toObjs(j.CookieJar.Cookies(u), rt)
}

// Set handles the receipt of the cookies in a reply for the given option.
func (j *CookieJar) Set(call goja.FunctionCall, rt *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		js.Throw(rt, errors.New("set first parameter must be url string"))
	}
	var cookies []*http.Cookie
	switch e := call.Argument(1).Export().(type) {
	case map[string]any:
		cookies = append(cookies, toCookie(e))
	case []any:
		for _, cookie := range cookies {
			cookies = append(cookies, toCookie(cast.ToStringMap(cookie)))
		}
	default:
		js.Throw(rt, errors.New("set second parameter must be cookie object"))
	}
	if len(cookies) == 0 {
		return goja.Undefined()
	}

	j.CookieJar.SetCookies(u, cookies)
	return goja.Undefined()
}

// Del handles the receipt of the cookies in a reply for the given URL.
func (j *CookieJar) Del(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		js.Throw(rt, err)
	}
	j.CookieJar.RemoveCookie(u)
	return goja.Undefined()
}

var sameSiteMapping = [...]string{
	http.SameSiteDefaultMode: "",
	http.SameSiteLaxMode:     "lax",
	http.SameSiteStrictMode:  "strict",
	http.SameSiteNoneMode:    "none",
}

func toObj(cookie *http.Cookie, rt *goja.Runtime) goja.Value {
	o := rt.NewObject()
	_ = o.Set("domain", rt.ToValue(cookie.Domain))
	_ = o.Set("expires", rt.ToValue(cookie.Expires.Unix()))
	_ = o.Set("name", rt.ToValue(cookie.Name))
	_ = o.Set("path", rt.ToValue(cookie.Path))
	_ = o.Set("sameSite", rt.ToValue(sameSiteMapping[cookie.SameSite]))
	_ = o.Set("secure", rt.ToValue(cookie.Secure))
	_ = o.Set("value", rt.ToValue(cookie.Value))
	_ = o.Set("toString", func(goja.FunctionCall) goja.Value {
		return rt.ToValue(cookie.String())
	})
	return o
}

func toObjs(cookies []*http.Cookie, rt *goja.Runtime) goja.Value {
	ret := make([]goja.Value, 0, len(cookies))
	for _, cookie := range cookies {
		ret = append(ret, toObj(cookie, rt))
	}
	return rt.ToValue(ret)
}

func toCookie(o map[string]any) *http.Cookie {
	var sameSite = http.SameSiteDefaultMode
	switch cast.ToString(o["sameSite"]) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	expires := cast.ToInt64(o["expires"])
	if expires == 0 {
		expires = time.Now().Add(time.Hour * 72).Unix()
	}
	return &http.Cookie{
		Domain:   cast.ToString(o["domain"]),
		Expires:  time.Unix(expires, 0),
		Name:     cast.ToString(o["name"]),
		Path:     cast.ToString(o["path"]),
		SameSite: sameSite,
		Value:    cast.ToString(o["value"]),
		MaxAge:   cast.ToInt(o["maxAge"]),
		Secure:   cast.ToBool(o["secure"]),
		HttpOnly: cast.ToBool(o["httpOnly"]),
	}
}
