package url

import (
	pkgurl "net/url"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("node:url", new(Module))
}

type Module struct{}

func (Module) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.NewObject()
	url, _ := new(URL).Instantiate(rt)
	_ = ret.Set("URL", url)
	urlSearchParams, _ := new(URLSearchParams).Instantiate(rt)
	_ = ret.Set("URLSearchParams", urlSearchParams)
	return ret, nil
}

func (Module) Global() {}

// URL is a component of the URL standard, which defines what constitutes
// a valid Uniform Resource Locator and the API that accesses and manipulates URLs.
// The URL standard also defines concepts such as domains, hosts, and IP addresses,
// and also attempts to describe in a standard way the legacy
// application/x-www-form-urlencoded MIME type used to submit web forms' contents
// as a set of key/value pairs.
// https://developer.mozilla.org/en-US/docs/Web/API/URL_API
type URL struct{}

func (u *URL) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()

	_ = p.DefineAccessorProperty("hash", rt.ToValue(u.hash), rt.ToValue(u.setHash), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("host", rt.ToValue(u.host), rt.ToValue(u.setHost), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("hostname", rt.ToValue(u.hostname), rt.ToValue(u.setHostname), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("href", rt.ToValue(u.href), rt.ToValue(u.setHref), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("origin", rt.ToValue(u.origin), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("password", rt.ToValue(u.password), rt.ToValue(u.setPassword), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("pathname", rt.ToValue(u.pathname), rt.ToValue(u.setPathname), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("port", rt.ToValue(u.port), rt.ToValue(u.setPort), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("protocol", rt.ToValue(u.protocol), rt.ToValue(u.setProtocol), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("username", rt.ToValue(u.username), rt.ToValue(u.setUsername), sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("search", rt.ToValue(u.search), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("searchParams", rt.ToValue(u.searchParams), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	_ = p.Set("canParse", u.canParse)
	_ = p.Set("createObjectURL", u.createObjectURL)
	_ = p.Set("revokeObjectURL", u.revokeObjectURL)
	_ = p.Set("parse", u.parse)
	_ = p.Set("toJSON", u.toJSON)

	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("URL") })
	_ = p.SetSymbol(sobek.SymHasInstance, func(call sobek.FunctionCall) sobek.Value { return rt.ToValue(call.Argument(0).ExportType() == typeURL) })
	return p
}

func (u *URL) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("URL constructor requires at least 1 argument"))
	}

	var (
		rawURL    = call.Argument(0).String()
		parsedURL *pkgurl.URL
		err       error
	)

	if base := call.Argument(1); !sobek.IsUndefined(base) {
		baseURL, err := pkgurl.Parse(base.String())
		if err != nil {
			js.Throw(rt, err)
		}
		parsedURL, err = baseURL.Parse(rawURL)
	} else {
		parsedURL, err = pkgurl.Parse(rawURL)
	}
	if err != nil {
		js.Throw(rt, err)
	}

	searchParams, err := js.New(rt, "URLSearchParams", rt.ToValue(parsedURL.RawQuery))
	if err != nil {
		js.Throw(rt, err)
	}

	obj := rt.ToValue(&url{
		url:          parsedURL,
		searchParams: searchParams,
	}).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (u *URL) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := u.prototype(rt)
	ctor := rt.ToValue(u.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

var (
	typeURL = reflect.TypeOf((*url)(nil))
)

func toURL(rt *sobek.Runtime, value sobek.Value) *url {
	if value.ExportType() == typeURL {
		return value.Export().(*url)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type URL`))
}

type url struct {
	url          *pkgurl.URL
	searchParams *sobek.Object
}

func (u *url) String() string {
	u.url.RawQuery = u.searchParams.String()
	return u.url.String()
}

func (*URL) hash(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	if this.url.Fragment == "" {
		return rt.ToValue("")
	}
	return rt.ToValue("#" + this.url.Fragment)
}

func (*URL) setHash(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	this.url.Fragment = strings.TrimPrefix(call.Argument(0).String(), "#")
	return sobek.Undefined()
}

func (*URL) host(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.url.Host)
}

func (*URL) setHost(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	this.url.Host = call.Argument(0).String()
	return sobek.Undefined()
}

func (*URL) hostname(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	host := this.url.Host
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}
	return rt.ToValue(host)
}

func (*URL) setHostname(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	hostname := call.Argument(0).String()
	if port := this.url.Port(); port != "" {
		this.url.Host = hostname + ":" + port
	} else {
		this.url.Host = hostname
	}
	return sobek.Undefined()
}

func (*URL) href(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.String())
}

func (*URL) setHref(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	newURL, err := pkgurl.Parse(call.Argument(0).String())
	if err != nil {
		js.Throw(rt, err)
	}
	this.url = newURL
	return sobek.Undefined()
}

func (*URL) origin(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.url.Scheme + "://" + this.url.Host)
}

func (*URL) password(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	if this.url.User == nil {
		return rt.ToValue("")
	}
	if pass, ok := this.url.User.Password(); ok {
		return rt.ToValue(pass)
	}
	return rt.ToValue("")
}

func (*URL) setPassword(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	password := call.Argument(0).String()
	username := ""
	if this.url.User != nil {
		username = this.url.User.Username()
	}
	this.url.User = pkgurl.UserPassword(username, password)
	return sobek.Undefined()
}

func (*URL) pathname(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.url.Path)
}

func (*URL) setPathname(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	this.url.Path = call.Argument(0).String()
	return sobek.Undefined()
}

func (*URL) port(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.url.Port())
}

func (*URL) setPort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	port := call.Argument(0).String()
	host := this.url.Hostname()
	if port != "" {
		this.url.Host = host + ":" + port
	} else {
		this.url.Host = host
	}
	return sobek.Undefined()
}

func (*URL) protocol(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.url.Scheme + ":")
}

func (*URL) setProtocol(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	protocol := call.Argument(0).String()
	if strings.HasSuffix(protocol, ":") {
		protocol = protocol[:len(protocol)-1]
	}
	this.url.Scheme = protocol
	return sobek.Undefined()
}

func (*URL) username(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	if this.url.User == nil {
		return rt.ToValue("")
	}
	return rt.ToValue(this.url.User.Username())
}

func (*URL) setUsername(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	username := call.Argument(0).String()
	password := ""
	if this.url.User != nil {
		if pass, ok := this.url.User.Password(); ok {
			password = pass
		}
	}
	this.url.User = pkgurl.UserPassword(username, password)
	return sobek.Undefined()
}

func (*URL) search(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	search := this.searchParams.String()
	if len(search) > 0 {
		search = "?" + search
	}
	return rt.ToValue(search)
}

func (*URL) searchParams(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return this.searchParams
}

func (*URL) parse(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("URL parse requires at least 1 argument"))
	}

	var (
		rawURL    = call.Argument(0).String()
		parsedURL *pkgurl.URL
		err       error
	)

	if base := call.Argument(1); !sobek.IsUndefined(base) {
		baseURL, err := pkgurl.Parse(base.String())
		if err != nil {
			return sobek.Null()
		}
		parsedURL, err = baseURL.Parse(rawURL)
	} else {
		parsedURL, err = pkgurl.Parse(rawURL)
	}
	if err != nil {
		return sobek.Null()
	}

	searchParams, err := js.New(rt, "URLSearchParams", rt.ToValue(parsedURL.RawQuery))
	if err != nil {
		return sobek.Null()
	}

	obj := rt.ToValue(&url{
		url:          parsedURL,
		searchParams: searchParams,
	}).(*sobek.Object)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

func (*URL) canParse(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("URL canParse requires at least 1 argument"))
	}

	var (
		rawURL = call.Argument(0).String()
		err    error
	)

	if base := call.Argument(1); !sobek.IsUndefined(base) {
		baseURL, err := pkgurl.Parse(base.String())
		if err != nil {
			return rt.ToValue(false)
		}
		_, err = baseURL.Parse(rawURL)
	} else {
		_, err = pkgurl.Parse(rawURL)
	}
	return rt.ToValue(err == nil)
}

func (*URL) createObjectURL(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	panic(rt.NewTypeError("URL createObjectURL not implemented"))
}

func (*URL) revokeObjectURL(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	panic(rt.NewTypeError("URL createObjectURL not implemented"))
}

func (*URL) toJSON(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURL(rt, call.This)
	return rt.ToValue(this.String())
}
