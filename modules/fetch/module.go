package fetch

import (
	"errors"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/modules"
)

func init() {
	jar := NewCookieJar()
	fetch := ski.NewFetch().(*http.Client)
	fetch.Jar = jar
	modules.Register("http", &Http{fetch})
	modules.Register("fetch", &Module{fetch, jar})
}

type Module struct {
	ski.Fetch
	CookieJar
}

func (m *Module) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if m.Fetch == nil {
		return nil, errors.New("fetch can not be nil")
	}
	if m.CookieJar == nil {
		return nil, errors.New("CookieJar can not nil")
	}
	ret := rt.NewObject()
	cookieJar, _ := (&CookieJarModule{m.CookieJar}).Instantiate(rt)
	_ = ret.Set("cookieJar", cookieJar)
	fetch, _ := (&Fetch{m.Fetch}).Instantiate(rt)
	_ = ret.Set("fetch", fetch)
	request, _ := new(Request).Instantiate(rt)
	_ = ret.Set("Request", request)
	response, _ := new(Response).Instantiate(rt)
	_ = ret.Set("Response", response)
	headers, _ := new(Headers).Instantiate(rt)
	_ = ret.Set("Headers", headers)
	formData, _ := new(FormData).Instantiate(rt)
	_ = ret.Set("FormData", formData)
	abortController, _ := new(AbortController).Instantiate(rt)
	_ = ret.Set("AbortController", abortController)
	abortSignal, _ := new(AbortSignal).Instantiate(rt)
	_ = ret.Set("AbortSignal", abortSignal)
	return ret, nil
}

func (*Module) Global() {}
