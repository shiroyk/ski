package fetch

import (
	"errors"
	"net/http"
	pkgurl "net/url"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// CookieJarModule manages storage and use of cookies in HTTP requests.
type CookieJarModule struct{ CookieJar }

func (c *CookieJarModule) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("get", c.get)
	_ = p.Set("getAll", c.getAll)
	_ = p.Set("set", c.set)
	_ = p.Set("del", c.del)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("CookieJarModule") })
	return p
}

func (*CookieJarModule) constructor(_ sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	panic(rt.NewTypeError("Illegal constructor"))
}

func (c *CookieJarModule) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if c.CookieJar == nil {
		return nil, errors.New("CookieJar can not nil")
	}
	proto := c.prototype(rt)
	ctor := rt.ToValue(c.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (c *CookieJarModule) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	url := call.Argument(0)
	if sobek.IsUndefined(url) || sobek.IsNull(url) {
		js.Throw(rt, errors.New("get url must not be empty"))
	}
	u, err := pkgurl.Parse(url.String())
	if err != nil {
		js.Throw(rt, err)
	}

	cookies := c.Cookies(u)
	name := call.Argument(1)
	if sobek.IsUndefined(name) || sobek.IsNull(name) {
		js.Throw(rt, errors.New("get name must not be empty"))
	}
	n := name.String()
	for _, cookie := range cookies {
		if cookie.Name == n {
			return cookieToObject(cookie, rt.NewObject())
		}
	}

	return sobek.Null()
}

func (c *CookieJarModule) getAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	url := call.Argument(0)
	if sobek.IsUndefined(url) || sobek.IsNull(url) {
		js.Throw(rt, errors.New("getAll url must not be empty"))
	}
	u, err := pkgurl.Parse(url.String())
	if err != nil {
		js.Throw(rt, err)
	}

	cookies := c.Cookies(u)

	var name string
	if n := call.Argument(1); !sobek.IsUndefined(n) {
		name = n.String()
	}

	result := rt.NewArray()

	for _, cookie := range cookies {
		if name != "" && cookie.Name != name {
			continue
		}
		obj := cookieToObject(cookie, rt.NewObject())
		_ = result.Set(result.Get("length").String(), obj)
	}

	return result
}

func (c *CookieJarModule) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	url := call.Argument(0)
	if sobek.IsUndefined(url) || sobek.IsNull(url) {
		js.Throw(rt, errors.New("set url must not be empty"))
	}
	u, err := pkgurl.Parse(url.String())
	if err != nil {
		js.Throw(rt, err)
	}

	options := call.Argument(1).ToObject(rt)
	cookie := objectToCookie(options)
	c.SetCookies(u, []*http.Cookie{cookie})
	return sobek.Undefined()
}

func (c *CookieJarModule) del(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	url := call.Argument(0)
	if sobek.IsUndefined(url) || sobek.IsNull(url) {
		js.Throw(rt, errors.New("del url must not be empty"))
	}
	u, err := pkgurl.Parse(url.String())
	if err != nil {
		js.Throw(rt, err)
	}

	c.RemoveCookie(u)
	return sobek.Undefined()
}

var sameSiteMapping = [...]string{
	http.SameSiteDefaultMode: "",
	http.SameSiteLaxMode:     "lax",
	http.SameSiteStrictMode:  "strict",
	http.SameSiteNoneMode:    "none",
}

func cookieToObject(cookie *http.Cookie, obj *sobek.Object) *sobek.Object {
	_ = obj.Set("name", cookie.Name)
	_ = obj.Set("value", cookie.Value)
	if !cookie.Expires.IsZero() {
		_ = obj.Set("expires", cookie.Expires.UnixMilli())
	}
	_ = obj.Set("domain", cookie.Domain)
	_ = obj.Set("path", cookie.Path)
	_ = obj.Set("secure", cookie.Secure)
	_ = obj.Set("httpOnly", cookie.HttpOnly)
	if cookie.SameSite != http.SameSiteDefaultMode {
		_ = obj.Set("sameSite", sameSiteMapping[cookie.SameSite])
	}
	return obj
}

func objectToCookie(options *sobek.Object) *http.Cookie {
	cookie := new(http.Cookie)
	for _, key := range options.Keys() {
		value := options.Get(key)
		switch key {
		case "name":
			cookie.Name = value.String()
		case "value":
			cookie.Value = value.String()
		case "domain":
			cookie.Domain = value.String()
		case "path":
			cookie.Path = value.String()
		case "secure":
			cookie.Secure = value.ToBoolean()
		case "httpOnly":
			cookie.HttpOnly = value.ToBoolean()
		case "expires":
			cookie.Expires = time.UnixMilli(value.ToInteger())
		case "sameSite":
			switch value.String() {
			case "lax":
				cookie.SameSite = http.SameSiteLaxMode
			case "strict":
				cookie.SameSite = http.SameSiteStrictMode
			case "none":
				cookie.SameSite = http.SameSiteNoneMode
			}
		}
	}
	return cookie
}
