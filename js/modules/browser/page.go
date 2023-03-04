package browser

import (
	"github.com/dop251/goja"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
)

// Page the rod.Page wrap
type Page struct {
	page *rod.Page
}

// Activate (focuses) the page
func (p *Page) Activate() (*Page, error) {
	_, err := p.page.Activate()
	return p, err
}

// AddScriptTag to page. If url is empty, content will be used.
func (p *Page) AddScriptTag(url, content string) error {
	return p.page.AddScriptTag(url, content)
}

// AddStyleTag to page. If url is empty, content will be used.
func (p *Page) AddStyleTag(url, content string) error {
	return p.page.AddStyleTag(url, content)
}

// Close tries to close page, running its beforeunload hooks, if has any.
func (p *Page) Close() error {
	return p.page.Close()
}

// Element retries until an element in the page that matches the CSS selector, then returns
// the matched element.
func (p *Page) Element(selector string) *rod.Element {
	return p.page.MustElement(selector)
}

// Emulate the device, such as iPhone9. If device is devices.Clear, it will clear the override.
func (p *Page) Emulate(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	device := devices.Clear
	if !call.Argument(0).StrictEquals(goja.Undefined()) {
		device = jsonValue[devices.Device](call.Argument(0), vm)
	}
	p.page.MustEmulate(device)
	return
}

// Info of the page, such as the URL or title of the page
func (p *Page) Info() *proto.TargetTargetInfo {
	return p.page.MustInfo()
}
