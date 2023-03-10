package browser

import (
	"io"
	"time"

	"github.com/dop251/goja"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/shiroyk/cloudcat/js/common"
)

// Page the rod.Page mapping
type Page struct {
	*rod.Page
}

// NewPage creates a new Page mapping
func NewPage(p *rod.Page, vm *goja.Runtime) goja.Value {
	return vm.ToValue(mappingPage(Page{p}))
}

func mappingPage(page Page) map[string]any {
	return map[string]any{
		"activate":             page.Activate,
		"addScriptTag":         page.AddScriptTag,
		"addStyleTag":          page.AddStyleTag,
		"browser":              func() Browser { return Browser{page.Browser()} },
		"close":                page.Close,
		"cookies":              page.Cookies,
		"eachEvent":            page.EachEvent,
		"element":              page.Element,
		"elementByJS":          page.ElementByJS,
		"elementFromNode":      page.ElementFromNode,
		"elementFromObject":    page.ElementFromObject,
		"elementFromPoint":     page.ElementFromPoint,
		"elementR":             page.ElementR,
		"elements":             page.Elements,
		"elementsByJS":         page.ElementsByJS,
		"elementsX":            page.ElementsX,
		"elementX":             page.ElementX,
		"emulate":              page.Emulate,
		"eval":                 page.Eval,
		"evalOnNewDocument":    page.EvalOnNewDocument,
		"evaluate":             page.Evaluate,
		"getResource":          page.GetResource,
		"getSessionID":         page.GetSessionID,
		"getWindow":            page.GetWindow,
		"handleDialog":         page.HandleDialog,
		"handleFileDialog":     page.HandleFileDialog,
		"has":                  page.Has,
		"hasR":                 page.HasR,
		"hasX":                 page.HasX,
		"html":                 page.HTML,
		"info":                 page.Info,
		"insertText":           page.InsertText,
		"isIframe":             page.IsIframe,
		"keyActions":           page.KeyActions,
		"navigate":             page.Navigate,
		"navigateBack":         page.NavigateBack,
		"navigateForward":      page.NavigateForward,
		"objectToJSON":         page.ObjectToJSON,
		"overlay":              page.Overlay,
		"pdf":                  page.PDF,
		"release":              page.Release,
		"reload":               page.Reload,
		"screenshot":           page.Screenshot,
		"search":               page.Search,
		"setBlockedURLs":       page.SetBlockedURLs,
		"setCookies":           page.SetCookies,
		"setDocumentContent":   page.SetDocumentContent,
		"setExtraHeaders":      page.SetExtraHeaders,
		"setUserAgent":         page.SetUserAgent,
		"setViewport":          page.SetViewport,
		"setWindow":            page.SetWindow,
		"stopLoading":          page.StopLoading,
		"string":               page.String,
		"wait":                 page.Wait,
		"waitElementsMoreThan": page.WaitElementsMoreThan,
		"waitEvent":            page.WaitEvent,
		"waitIdle":             page.WaitIdle,
		"waitLoad":             page.WaitLoad,
		"waitNavigation":       page.WaitNavigation,
		"waitOpen":             page.WaitOpen,
		"waitRepaint":          page.WaitRepaint,
		"waitRequestIdle":      page.WaitRequestIdle,
	}
}

// EvalOptions for Page.Evaluate
type EvalOptions struct {
	ByValue      bool                       `json:"byValue"`
	AwaitPromise bool                       `json:"awaitPromise"`
	ThisObj      *proto.RuntimeRemoteObject `json:"thisObj"`
	JS           string                     `json:"js"`
	JSArgs       []any                      `json:"jsArgs"`
	UserGesture  bool                       `json:"userGesture"`
}

func (e *EvalOptions) toRodEvalOptions() *rod.EvalOptions {
	return &rod.EvalOptions{
		ByValue:      e.ByValue,
		AwaitPromise: e.AwaitPromise,
		ThisObj:      e.ThisObj,
		JS:           e.JS,
		JSArgs:       e.JSArgs,
		UserGesture:  e.UserGesture,
	}
}

// Activate (focuses) the page
func (p *Page) Activate() (any, error) {
	page, err := p.Page.Activate()
	if err != nil {
		return nil, err
	}
	return mappingPage(Page{page}), nil
}

// Cookies returns the page cookies. By default it will return the cookies for current page.
// The urls is the list of URLs for which applicable cookies will be fetched.
func (p *Page) Cookies(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	urls := call.Argument(0).Export().([]string)
	cookies, err := p.Page.Cookies(urls)
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(cookies, vm)
}

// Element retries until an element in the page that matches the CSS selector, then returns
// the matched element.
func (p *Page) Element(selector string) (any, error) {
	element, err := p.Page.Element(selector)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// ElementByJS returns the element from the return value of the js function.
// If sleeper is nil, no retry will be performed.
// By default, it will retry until the js function doesn't return null.
// To customize the retry logic, check the examples of Page.Sleeper.
func (p *Page) ElementByJS(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	element, err := p.Page.ElementByJS(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return NewElement(element, vm)
}

// ElementFromNode creates an Element from the node, NodeID or BackendNodeID must be specified.
func (p *Page) ElementFromNode(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.DOMNode](call.Argument(0), vm)
	element, err := p.Page.ElementFromNode(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return NewElement(element, vm)
}

// ElementFromObject creates an Element from the remote object id.
func (p *Page) ElementFromObject(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.RuntimeRemoteObject](call.Argument(0), vm)
	element, err := p.Page.ElementFromObject(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return NewElement(element, vm)
}

// ElementFromPoint creates an Element from the absolute point on the page.
// The point should include the window scroll offset.
func (p *Page) ElementFromPoint(x, y int) (any, error) {
	element, err := p.Page.ElementFromPoint(x, y)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// ElementR retries until an element in the page that matches the css selector and it's text matches the jsRegex,
// then returns the matched element.
func (p *Page) ElementR(selector, jsRegex string) (any, error) {
	element, err := p.Page.ElementR(selector, jsRegex)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// Elements returns all elements that match the css selector
func (p *Page) Elements(selector string) (Elements, error) {
	elements, err := p.Page.Elements(selector)
	if err != nil {
		return nil, err
	}
	return NewElements(elements), nil
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	elements, err := p.Page.ElementsByJS(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(NewElements(elements))
}

// ElementsX returns all elements that match the XPath selector
func (p *Page) ElementsX(xpath string) (Elements, error) {
	elements, err := p.Page.ElementsX(xpath)
	if err != nil {
		return nil, err
	}
	return NewElements(elements), nil
}

// ElementX retries until an element in the page that matches one of the XPath selectors, then returns
// the matched element.
func (p *Page) ElementX(xPath string) (any, error) {
	element, err := p.Page.ElementX(xPath)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// Evaluate js on the page.
func (p *Page) Evaluate(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	res, err := p.Page.Evaluate(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(res)
}

// Emulate the device, such as iPhone9. If device is devices.Clear, it will clear the override.
func (p *Page) Emulate(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	device := devices.Clear
	if !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
		device = toGoStruct[devices.Device](call.Argument(0), vm)
	}
	p.MustEmulate(device)
	return
}

// Eval is a shortcut for Page.Evaluate with AwaitPromise, ByValue set to true.
func (p *Page) Eval(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	js := call.Argument(0).String()
	args := make([]any, 0, len(call.Arguments)-1)
	for _, value := range call.Arguments[1:] {
		args = append(args, value)
	}

	value, err := p.Page.Eval(js, args...)
	if err != nil {
		common.Throw(vm, err)
	}

	return toJSObject(value, vm)
}

// GetResource content by the url. Such as image, css, html, etc.
// Use the proto.PageGetResourceTree to list all the resources.
func (p *Page) GetResource(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	resource, err := p.Page.GetResource(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(vm.NewArrayBuffer(resource))
}

// GetWindow position and size info
func (p *Page) GetWindow(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	window, err := p.Page.GetWindow()
	if err != nil {
		common.Throw(vm, err)
	}

	return toJSObject(window, vm)
}

// HandleDialog accepts or dismisses next JavaScript initiated dialog (alert, confirm, prompt, or onbeforeunload).
// Because modal dialog will block js, usually you have to trigger the dialog in another goroutine.
// For example:
//
//	const { wait, handle } = page.handleDialog()
//	page.element("button").click()
//	wait()
//	handle(true, "")
func (p *Page) HandleDialog(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	waitNative, handleNative := p.Page.MustHandleDialog()
	obj := vm.NewObject()
	_ = obj.Set("wait", func() any { return toJSObject(waitNative(), vm) })
	_ = obj.Set("handle", handleNative)
	return obj
}

// Has an element that matches the css selector
func (p *Page) Has(selector string) (bool, any, error) {
	has, e, err := p.Page.Has(selector)
	if err != nil {
		return false, nil, err
	}
	return has, mappingElement(Element{e}), nil
}

// HasR an element that matches the css selector and its display text matches the jsRegex.
func (p *Page) HasR(selector, jsRegex string) (bool, any, error) {
	has, e, err := p.Page.HasR(selector, jsRegex)
	if err != nil {
		return false, nil, err
	}
	return has, mappingElement(Element{e}), nil
}

// HasX an element that matches the XPath selector
func (p *Page) HasX(selector string) (bool, any, error) {
	has, e, err := p.Page.HasX(selector)
	if err != nil {
		return false, nil, err
	}
	return has, mappingElement(Element{e}), nil
}

// Info of the page, such as the URL or title of the page
func (p *Page) Info(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	info, err := p.Page.Info()
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(info, vm)
}

// ObjectToJSON by object id
func (p *Page) ObjectToJSON(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.RuntimeRemoteObject](call.Argument(0), vm)
	value, err := p.Page.ObjectToJSON(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(value)
}

// PDF prints page as PDF
func (p *Page) PDF(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.PagePrintToPDF](call.Argument(0), vm)
	value, err := p.Page.PDF(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	buf, err := io.ReadAll(value)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(vm.NewArrayBuffer(buf))
}

// Release the remote object. Usually, you don't need to call it.
// When a page is closed or reloaded, all remote objects will be released automatically.
// It's useful if the page never closes or reloads.
func (p *Page) Release(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.RuntimeRemoteObject](call.Argument(0), vm)
	if err := p.Page.Release(&target); err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// Screenshot captures the screenshot of current page.
func (p *Page) Screenshot(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	var fullPage bool
	if !goja.IsUndefined(call.Argument(0)) {
		fullPage = call.Argument(0).ToBoolean()
	}
	target := toGoStruct[proto.PageCaptureScreenshot](call.Argument(1), vm)
	screenshot, err := p.Page.Screenshot(fullPage, &target)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(vm.NewArrayBuffer(screenshot))
}

// Search for the given query in the DOM tree until the result count is not zero, before that it will keep retrying.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) Search(query string) (any, error) {
	result, err := p.Page.Search(query)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"first":       Element{result.First},
		"searchID":    result.SearchID,
		"resultCount": result.ResultCount,
		"get": func(i, l int) (Elements, error) {
			elements, err := result.Get(i, l)
			if err != nil {
				return nil, err
			}
			return NewElements(elements), nil
		},
		"all": func() (Elements, error) {
			elements, err := result.All()
			if err != nil {
				return nil, err
			}
			return NewElements(elements), nil
		},
		"release": result.Release,
	}, nil
}

// SetCookies is similar to Browser.SetCookies .
func (p *Page) SetCookies(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[[]*proto.NetworkCookieParam](call.Argument(0), vm)
	err := p.Page.SetCookies(target)
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// SetUserAgent (browser brand, accept-language, etc) of the page.
// If req is nil, a default user agent will be used, a typical mac chrome.
func (p *Page) SetUserAgent(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.NetworkSetUserAgentOverride](call.Argument(0), vm)
	err := p.Page.SetUserAgent(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// SetViewport overrides the values of device screen dimensions
func (p *Page) SetViewport(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.EmulationSetDeviceMetricsOverride](call.Argument(0), vm)
	err := p.Page.SetViewport(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// SetWindow location and size
func (p *Page) SetWindow(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[proto.BrowserBounds](call.Argument(0), vm)
	err := p.Page.SetWindow(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// Wait until the js returns true
func (p *Page) Wait(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	err := p.Page.Wait(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// WaitIdle waits until the next window.requestIdleCallback is called.
func (p *Page) WaitIdle(timeout string) (err error) {
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return err
	}
	return p.Page.WaitIdle(duration)
}

// WaitOpen waits for the next new page opened by the current one
func (p *Page) WaitOpen() func() (any, error) {
	return func() (any, error) {
		page, err := p.Page.WaitOpen()()
		if err != nil {
			return Page{}, err
		}
		return mappingPage(Page{page}), nil
	}
}
