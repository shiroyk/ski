package browser

import (
	"github.com/dop251/goja"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/shiroyk/cloudcat/js/common"
)

// Element represents the DOM element
type Element struct {
	*rod.Element
}

// Elements provides some helpers to deal with element list
type Elements []map[string]any

// First returns the first element, if the list is empty returns nil
func (els Elements) First() any {
	if els.Empty() {
		return nil
	}
	return &(els[0])
}

// Last returns the last element, if the list is empty returns nil
func (els Elements) Last() any {
	if els.Empty() {
		return nil
	}
	return &(els[len(els)-1])
}

// Empty returns true if the list is empty
func (els Elements) Empty() bool {
	return len(els) == 0
}

// NewElement creates a new Element mapping
func NewElement(ele *rod.Element, vm *goja.Runtime) goja.Value {
	return vm.ToValue(mappingElement(Element{ele}))
}

// NewElements creates a new Elements mapping
func NewElements(elements rod.Elements) Elements {
	mapping := make(Elements, 0, len(elements))
	for _, element := range elements {
		mapping = append(mapping, mappingElement(Element{element}))
	}
	return mapping
}

func mappingElement(element Element) map[string]any {
	return map[string]any{
		"attribute":        element.Attribute,
		"backgroundImage":  element.BackgroundImage,
		"blur":             element.Blur,
		"click":            element.Click,
		"containsElement":  element.ContainsElement,
		"describe":         element.Describe,
		"disabled":         element.Disabled,
		"element":          element.NElement,
		"elementByJS":      element.ElementByJS,
		"elementR":         element.ElementR,
		"elements":         element.Elements,
		"elementsByJS":     element.ElementsByJS,
		"elementsX":        element.ElementsX,
		"equal":            element.Equal,
		"eval":             element.Eval,
		"evaluate":         element.Evaluate,
		"focus":            element.Focus,
		"frame":            element.Frame,
		"getSessionID":     element.GetSessionID,
		"getXPath":         element.GetXPath,
		"has":              element.Has,
		"hasR":             element.HasR,
		"hasX":             element.HasX,
		"hover":            element.Hover,
		"html":             element.HTML,
		"input":            element.Input,
		"inputTime":        element.InputTime,
		"interactable":     element.Interactable,
		"keyActions":       element.KeyActions,
		"matches":          element.Matches,
		"moveMouseOut":     element.MoveMouseOut,
		"next":             element.Next,
		"overlay":          element.Overlay,
		"page":             element.Page,
		"parent":           element.Parent,
		"parents":          element.Parents,
		"previous":         element.Previous,
		"property":         element.Property,
		"release":          element.Release,
		"remove":           element.Remove,
		"resource":         element.Resource,
		"screenshot":       element.Screenshot,
		"scrollIntoView":   element.ScrollIntoView,
		"select":           element.Select,
		"selectAllText":    element.SelectAllText,
		"selectText":       element.SelectText,
		"setFiles":         element.SetFiles,
		"shadowRoot":       element.ShadowRoot,
		"shape":            element.Shape,
		"string":           element.String,
		"tap":              element.Tap,
		"text":             element.Text,
		"type":             element.Type,
		"visible":          element.Visible,
		"wait":             element.Wait,
		"waitEnabled":      element.WaitEnabled,
		"waitInteractable": element.WaitInteractable,
		"waitInvisible":    element.WaitInvisible,
		"waitLoad":         element.WaitLoad,
		"waitStable":       element.WaitStable,
		"waitStableRAF":    element.WaitStableRAF,
		"waitVisible":      element.WaitVisible,
		"waitWritable":     element.WaitWritable,
	}
}

// BackgroundImage returns the css background-image of the element
func (el *Element) BackgroundImage(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	image, err := el.Element.BackgroundImage()
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(vm.NewArrayBuffer(image))
}

// ContainsElement check if the target is equal or inside the element.
func (el *Element) ContainsElement(element goja.Value) (bool, error) {
	value := element.Export().(Element)
	return el.Element.ContainsElement(value.Element)
}

// Describe the current element. The depth is the maximum depth at which children should be retrieved, defaults to 1,
// use -1 for the entire subtree or provide an integer larger than 0.
// The pierce decides whether or not iframes and shadow roots should be traversed when returning the subtree.
// The returned proto.DOMNode.NodeID will always be empty, because NodeID is not stable (when proto.DOMDocumentUpdated
// is fired all NodeID on the page will be reassigned to another value)
// we don't recommend using the NodeID, instead, use the BackendNodeID to identify the element.
func (el *Element) Describe(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	depth := call.Argument(0).ToInteger()
	pierce := call.Argument(1).ToBoolean()
	describe, err := el.Element.Describe(int(depth), pierce)
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(describe, vm)
}

// NElement returns the first child that matches the css selector
func (el *Element) NElement(selector string) (any, error) {
	element, err := el.Element.Element(selector)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// ElementByJS returns the element from the return value of the js
func (el *Element) ElementByJS(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[rod.EvalOptions](call.Argument(0), vm)
	element, err := el.Element.ElementByJS(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return NewElement(element, vm)
}

// ElementR returns the first child element that matches the css selector and its text matches the jsRegex.
func (el *Element) ElementR(selector, jsRegex string) (any, error) {
	element, err := el.Element.ElementR(selector, jsRegex)
	if err != nil {
		return nil, err
	}
	return mappingElement(Element{element}), nil
}

// Elements returns all elements that match the css selector
func (el *Element) Elements(selector string) (any, error) {
	elements, err := el.Element.Elements(selector)
	if err != nil {
		return nil, err
	}
	return NewElements(elements), nil
}

// ElementsByJS returns the elements from the return value of the js
func (el *Element) ElementsByJS(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[rod.EvalOptions](call.Argument(0), vm)
	elements, err := el.Element.ElementsByJS(&target)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(NewElements(elements))
}

// ElementsX returns all elements that match the XPath selector
func (el *Element) ElementsX(xpath string) (any, error) {
	elements, err := el.Element.ElementsX(xpath)
	if err != nil {
		return nil, err
	}
	return NewElements(elements), nil
}

// Equal checks if the two elements are equal.
func (el *Element) Equal(elm goja.Value) (bool, error) {
	value := elm.Export().(Element)
	return el.Element.Equal(value.Element)
}

// Eval is a shortcut for Element.Evaluate with AwaitPromise, ByValue and AutoExp set to true.
func (el *Element) Eval(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	js := call.Argument(0).String()
	args := make([]any, 0, len(call.Arguments)-1)
	for _, value := range call.Arguments[1:] {
		args = append(args, value)
	}

	value, err := el.Element.Eval(js, args...)
	if err != nil {
		common.Throw(vm, err)
	}

	return toJSObject(value, vm)
}

// Evaluate is just a shortcut of Page.Evaluate with This set to current element.
func (el *Element) Evaluate(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	res, err := el.Element.Evaluate(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(res)
}

// Frame creates a page instance that represents the iframe
func (el *Element) Frame() any {
	return mappingPage(Page{el.Element.MustFrame()})
}

// Has an element that matches the css selector
func (el *Element) Has(selector string) (bool, any, error) {
	has, e, err := el.Element.Has(selector)
	if err != nil {
		return false, nil, err
	}
	return has, mappingElement(Element{e}), nil
}

// HasR an element that matches the css selector and its display text matches the jsRegex.
func (el *Element) HasR(selector, jsRegex string) (bool, any, error) {
	has, e, err := el.Element.HasR(selector, jsRegex)
	if err != nil {
		return false, Element{}, err
	}
	return has, mappingElement(Element{e}), nil
}

// HasX an element that matches the XPath selector
func (el *Element) HasX(selector string) (bool, any, error) {
	has, e, err := el.Element.HasX(selector)
	if err != nil {
		return false, nil, err
	}
	return has, mappingElement(Element{e}), nil
}

// Interactable checks if the element is interactable with cursor.
// The cursor can be mouse, finger, stylus, etc.
// If not interactable err will be ErrNotInteractable, such as when covered by a modal,
func (el *Element) Interactable(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	interactable, err := el.Element.Interactable()
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(interactable, vm)
}

// Next returns the next sibling element in the DOM tree
func (el *Element) Next() any {
	return mappingElement(Element{el.Element.MustNext()})
}

// Page of the element
func (el *Element) Page() any {
	return mappingPage(Page{el.Element.Page()})
}

// Parent returns the parent element in the DOM tree
func (el *Element) Parent() any {
	return mappingElement(Element{el.Element.MustParent()})
}

// Parents that match the selector
func (el *Element) Parents(selector string) any {
	return NewElements(el.Element.MustParents(selector))
}

// Previous returns the previous sibling element in the DOM tree
func (el *Element) Previous() any {
	return mappingElement(Element{el.Element.MustPrevious()})
}

// Resource returns the "src" content of current element. Such as the jpg of <img src="a.jpg">
func (el *Element) Resource(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	image, err := el.Element.Resource()
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(vm.NewArrayBuffer(image))
}

// Screenshot of the area of the element
func (el *Element) Screenshot(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	var quality int
	if !goja.IsUndefined(call.Argument(1)) {
		quality = int(call.Argument(1).ToInteger())
	}
	target := toGoStruct[proto.PageCaptureScreenshotFormat](call.Argument(0), vm)
	screenshot, err := el.Element.Screenshot(target, quality)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(vm.NewArrayBuffer(screenshot))
}

// ShadowRoot returns the shadow root of this element
func (el *Element) ShadowRoot() any {
	return mappingElement(Element{el.Element.MustShadowRoot()})
}

// Shape of the DOM element content. The shape is a group of 4-sides polygons.
// A 4-sides polygon is not necessary a rectangle. 4-sides polygons can be apart from each other.
// For example, we use 2 4-sides polygons to describe the shape below:
//
//	  ____________          ____________
//	 /        ___/    =    /___________/    +     _________
//	/________/                                   /________/
func (el *Element) Shape(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	shape, err := el.Element.Shape()
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(shape, vm)
}

// Wait until the js returns true
func (el *Element) Wait(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	target := toGoStruct[EvalOptions](call.Argument(0), vm)
	err := el.Element.Wait(target.toRodEvalOptions())
	if err != nil {
		common.Throw(vm, err)
	}
	return goja.Undefined()
}

// WaitInteractable waits for the element to be interactable.
// It will try to scroll to the element on each try.
func (el *Element) WaitInteractable(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	interactable, err := el.Element.WaitInteractable()
	if err != nil {
		common.Throw(vm, err)
	}
	return toJSObject(interactable, vm)
}
