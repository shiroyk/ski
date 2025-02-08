package gq

import (
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	htmlutil "github.com/shiroyk/ski/modules/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// attr gets the value of an attribute for the first element in the set of matched elements.
func (Gq) attr(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("attr requires at least 1 argument"))
	}
	val, ok := sel.Attr(call.Argument(0).String())
	if !ok {
		return sobek.Null()
	}
	return rt.ToValue(val)
}

// href gets the href attribute's value, if URL is not absolute returns the absolute URL.
func (Gq) href(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	var base string
	if v := call.Argument(0); !sobek.IsUndefined(v) {
		base = v.String()
	}

	get := func(node *html.Node) (string, bool) {
		i := slices.IndexFunc(node.Attr, func(attr html.Attribute) bool { return attr.Key == "href" })
		if i < 0 {
			return "", false
		}

		href := node.Attr[i].Val
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			return href, true
		}
		if len(base) > 0 {
			if !strings.HasPrefix(href, ".") {
				if !strings.HasPrefix(href, "/") {
					href = "/" + href
				}
				return strings.TrimSuffix(base, "/") + href, true
			}
			hrefURL, err := url.Parse(href)
			if err != nil {
				js.Throw(rt, err)
			}
			baseURL, err := url.Parse(base)
			if err != nil {
				js.Throw(rt, err)
			}
			href = baseURL.ResolveReference(hrefURL).String()
		}
		return href, true
	}

	switch len(sel.Nodes) {
	case 0:
		return sobek.Null()
	case 1:
		s, ok := get(sel.Nodes[0])
		if !ok {
			return sobek.Null()
		}
		return rt.ToValue(s)
	default:
		ret := make([]string, 0, len(sel.Nodes))
		for _, node := range sel.Nodes {
			s, ok := get(node)
			if ok {
				ret = append(ret, s)
			}
		}
		return rt.ToValue(ret)
	}
}

// removeAttr remove an attribute from each element in the set of matched elements..
func (Gq) removeAttr(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("removeAttr requires at least 1 argument"))
	}
	ret := rt.ToValue(sel.RemoveAttr(call.Argument(0).String())).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// val gets the current value of the first element in the set of matched elements.
func (Gq) val(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if sel.Length() == 0 {
		return sobek.Undefined()
	}
	node := sel.Nodes[0]
	switch node.DataAtom {
	case atom.Input, atom.Textarea:
		val, ok := sel.Attr("value")
		if !ok {
			return sobek.Undefined()
		}
		return rt.ToValue(val)
	case atom.Select:
		nodes := sel.Find("option:checked")
		if nodes.Length() == 0 {
			return sobek.Undefined()
		}
		_, ok := sel.Attr("multiple")
		if ok {
			return rt.ToValue(goquery.Map(nodes, func(i int, s *goquery.Selection) string {
				return s.Text()
			}))
		}
		return rt.ToValue(nodes.First().Text())
	default:
		return sobek.Undefined()
	}
}

// html gets the HTML contents of the first element in the set of matched elements.
func (Gq) html(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	ret, err := sel.Html()
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(ret)
}

// text gets the combined text contents of each element in the set of matched elements,
// including their descendants.
func (Gq) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	return rt.ToValue(sel.Text())
}

// addClass adds the specified class(es) to each element in the set of matched elements.
func (Gq) addClass(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("addClass requires at least 1 argument"))
	}
	className := call.Argument(0)
	callback, ok := sobek.AssertFunction(className)
	if !ok {
		switch className.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(className, &class)
			sel.AddClass(class...)
		default:
			sel.AddClass(className.String())
		}
		return rt.ToValue(&gq{sel})
	}

	prototype := call.This.ToObject(rt).Prototype()
	for i, s := range sel.EachIter() {
		value := rt.ToValue(&gq{s}).(*sobek.Object)
		_ = value.SetPrototype(prototype)
		v, err := callback(value, rt.ToValue(i), value)
		if err != nil {
			js.Throw(rt, err)
		}
		switch v.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(v, &class)
			s.AddClass(class...)
		default:
			s.AddClass(v.String())
		}
	}

	return rt.ToValue(&gq{sel})
}

// hasClass determine whether any of the matched elements are assigned the given class.
func (Gq) hasClass(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("hasClass requires at least 1 argument"))
	}
	return rt.ToValue(sel.HasClass(call.Argument(0).String()))
}

// removeClass remove a single class or multiple classes from each element in the set of matched elements.
func (Gq) removeClass(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("removeClass requires at least 1 argument"))
	}

	className := call.Argument(0)
	callback, ok := sobek.AssertFunction(className)
	if !ok {
		switch className.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(className, &class)
			sel.RemoveClass(class...)
		default:
			sel.RemoveClass(className.String())
		}
		return rt.ToValue(&gq{sel})
	}

	prototype := call.This.ToObject(rt).Prototype()
	for i, s := range sel.EachIter() {
		value := rt.ToValue(&gq{s}).(*sobek.Object)
		_ = value.SetPrototype(prototype)
		v, err := callback(value, rt.ToValue(i), value)
		if err != nil {
			js.Throw(rt, err)
		}
		switch v.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(v, &class)
			s.RemoveClass(class...)
		default:
			s.RemoveClass(v.String())
		}
	}

	return rt.ToValue(&gq{sel})
}

// toggleClass add or remove one or more classes from each element in the set of matched elements,
// depending on either the class's presence or the value of the state argument.
func (Gq) toggleClass(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("toggleClass requires at least 1 argument"))
	}

	className := call.Argument(0)
	add := call.Argument(1).ToBoolean()
	modify := sel.RemoveClass
	if add {
		modify = sel.AddClass
	}
	callback, ok := sobek.AssertFunction(className)
	if !ok {
		switch className.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(className, &class)
			modify(class...)
		default:
			modify(className.String())
		}
		return rt.ToValue(&gq{sel})
	}

	prototype := call.This.ToObject(rt).Prototype()

	for i, s := range sel.EachIter() {
		val, ok := s.Attr("class")
		if !ok {
			continue
		}
		this := rt.ToValue(&gq{s}).(*sobek.Object)
		_ = this.SetPrototype(prototype)

		v, err := callback(this,
			rt.ToValue(i),
			rt.ToValue(val),
			rt.ToValue(add))
		if err != nil {
			js.Throw(rt, err)
		}
		modify := sel.RemoveClass
		if add {
			modify = sel.AddClass
		}
		switch v.ExportType().Kind() {
		case reflect.Slice, reflect.Array:
			var class []string
			_ = rt.ExportTo(v, &class)
			modify(class...)
		default:
			modify(v.String())
		}
	}

	return rt.ToValue(&gq{sel})
}

// get retrieve the html.Node elements matched.
func (Gq) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if idx := call.Argument(0); !sobek.IsUndefined(idx) {
		i := int(idx.ToInteger())
		if i > sel.Length() {
			return sobek.Null()
		}
		return rt.ToValue(sel.Get(i))
	}
	return rt.ToValue(sel.Nodes)
}

// index search for a given element from among the matched elements.
func (Gq) index(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		switch arg.ExportType() {
		case typeSelector:
			return rt.ToValue(sel.IndexMatcher(arg.Export().(*selector).sel))
		case typeSelection:
			return rt.ToValue(sel.IndexOfSelection(arg.Export().(*gq).sel))
		case htmlutil.TypeNode:
			return rt.ToValue(sel.IndexOfNode(arg.Export().(*html.Node)))
		default:
			return rt.ToValue(sel.IndexSelector(arg.String()))
		}
	}
	return rt.ToValue(sel.Index())
}

// toArray retrieve all the elements, as an array.
func (Gq) toArray(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	return rt.ToValue(sel.Nodes)
}
