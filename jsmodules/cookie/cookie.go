// Package cookie the cookie JS implementation
package cookie

import (
	"net/url"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Cookie{cloudcat.MustResolve[cloudcat.Cookie]()}
}

func init() {
	jsmodule.Register("cookie", &Module{})
}

// Cookie manages storage and use of cookies in HTTP requests.
type Cookie struct {
	cookie cloudcat.Cookie
}

// Get returns the cookies string for the given URL.
func (c *Cookie) Get(uri string) ([]string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return c.cookie.CookieString(u), nil
}

// Set handles the receipt of the cookies strung in a reply for the given URL.
func (c *Cookie) Set(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		js.Throw(vm, err)
	}

	str, err := js.ToStrings(call.Argument(1).Export())
	if err != nil {
		js.Throw(vm, err)
	}

	switch cookie := str.(type) {
	case string:
		c.cookie.SetCookies(u, cloudcat.ParseCookie(cookie))
	case []string:
		c.cookie.SetCookies(u, cloudcat.ParseSetCookie(cookie...))
	}
	return
}

// Del handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) Del(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	c.cookie.DeleteCookie(u)
	return nil
}
