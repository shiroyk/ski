package cookie

import (
	"net/url"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/modules"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Cookie{di.MustResolve[cache.Cookie]()}
}

func init() {
	modules.Register("cookie", &Module{})
}

// Cookie manages storage and use of cookies in HTTP requests.
type Cookie struct {
	cookie cache.Cookie
}

// Get returns the cookies string for the given URL.
func (c *Cookie) Get(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	return c.cookie.CookieString(u), nil
}

// Set handles the receipt of the cookies strung in a reply for the given URL.
func (c *Cookie) Set(uri, cookie string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	c.cookie.SetCookieString(u, cookie)
	return nil
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
