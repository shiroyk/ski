package gq

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
	htmlutil "github.com/shiroyk/ski/modules/html"
	"golang.org/x/net/html"
)

func init() {
	modules.Register("gq", new(Gq))
}

// Gq implements jQuery-like selector and traversing methods.
//
// usage:
//
//	import { default as $ } from "ski/gq";
//
//	export default function () {
//		return $('<div><span>ciallo</span></div>').find('span').text();
//	}
type Gq struct{}

func (g Gq) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	selection := new(goquery.Selection)
	prototype := call.This.Prototype()
	sel := call.Argument(0)
	context := call.Argument(1)
	if sobek.IsUndefined(sel) {
		goto RET
	}

	switch sel.ExportType() {
	case typeSelector:
		ctx := toSelection(rt, context)
		selection = ctx.FindMatcher(sel.Export().(*selector).sel)
	case htmlutil.TypeNode:
		selection = goquery.NewDocumentFromNode(sel.Export().(*html.Node)).Selection
	case htmlutil.TypeNodes:
		selection = nodesToSel(sel.Export().([]*html.Node))
	case typeSelection:
		selection = sel.Export().(*goquery.Selection)
	default:
		str := strings.TrimSpace(sel.String())
		if len(str) > 3 && str[0] == '<' && str[len(str)-1] == '>' {
			selection = toSelection(rt, sel)
			goto RET
		}

		ctx := toSelection(rt, context)
		selection = ctx.Find(str)
	}

RET:
	ret := rt.ToValue(&gq{selection}).(*sobek.Object)
	_ = ret.SetPrototype(prototype)
	return ret
}

func (g Gq) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ctor := rt.ToValue(g.constructor).ToObject(rt)
	p := g.prototype(rt)
	_ = ctor.SetPrototype(p)
	_ = ctor.Set("prototype", p)
	_ = ctor.Set("selector", g.selector)
	_ = ctor.Set("parseHtml", g.parseHtml)
	return ctor, nil
}

func (g Gq) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("selector", g.selector)
	_ = p.Set("parseHtml", g.parseHtml)
	_ = p.Set("clone", g.clone)
	_ = p.Set("get", g.get)
	_ = p.Set("index", g.index)
	_ = p.Set("toArray", g.toArray)
	_ = p.DefineAccessorProperty("length", rt.ToValue(g.length), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.SetSymbol(sobek.SymIterator, g.values)

	// traversal
	_ = p.Set("find", g.find)
	_ = p.Set("children", g.children)
	_ = p.Set("parent", g.parent)
	_ = p.Set("parents", g.parents)
	_ = p.Set("next", g.next)
	_ = p.Set("prev", g.prev)
	_ = p.Set("siblings", g.siblings)
	_ = p.Set("nextAll", g.nextAll)
	_ = p.Set("prevAll", g.prevAll)
	_ = p.Set("nextUntil", g.nextUntil)
	_ = p.Set("prevUntil", g.prevUntil)
	_ = p.Set("parentsUntil", g.parentsUntil)
	_ = p.Set("closest", g.closest)
	_ = p.Set("contents", g.contents)

	// filter
	_ = p.Set("eq", g.eq)
	_ = p.Set("not", g.not)
	_ = p.Set("add", g.add)
	_ = p.Set("filter", g.filter)
	_ = p.Set("first", g.first)
	_ = p.Set("last", g.last)
	_ = p.Set("has", g.has)
	_ = p.Set("is", g.is)
	_ = p.Set("even", g.even)
	_ = p.Set("odd", g.odd)
	_ = p.Set("slice", g.slice)
	_ = p.Set("map", g.map_)
	_ = p.Set("each", g.each)

	// property
	_ = p.Set("attr", g.attr)
	_ = p.Set("prop", g.attr)
	_ = p.Set("text", g.text)
	_ = p.Set("val", g.val)
	_ = p.Set("html", g.html)
	_ = p.Set("href", g.href)
	_ = p.Set("removeAttr", g.removeAttr)
	_ = p.Set("removeProp", g.removeAttr)
	_ = p.Set("addClass", g.addClass)
	_ = p.Set("hasClass", g.hasClass)
	_ = p.Set("removeClass", g.removeClass)
	_ = p.Set("toggleClass", g.toggleClass)

	return p
}

func (Gq) selector(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	s, err := cascadia.Compile(call.Argument(0).String())
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(&selector{s})
}

func (Gq) parseHtml(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("parseHtml requires at least 1 argument"))
	}
	data := call.Argument(0).String()

	if ctx := call.Argument(1); !sobek.IsUndefined(ctx) {
		if ctx.ExportType() != htmlutil.TypeNode {
			panic(rt.NewTypeError("parseHtml context must be a html.Node"))
		}
		opt := html.ParseOptionEnableScripting(call.Argument(2).ToBoolean())
		nodes, err := html.ParseFragmentWithOptions(strings.NewReader(data), ctx.Export().(*html.Node), opt)
		if err != nil {
			js.Throw(rt, err)
		}
		return rt.ToValue(nodes)
	}

	node, err := htmlutil.Parse(data)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(node)
}

func (Gq) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	ret := rt.ToValue(&gq{sel.Clone()}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

func (Gq) length(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	return rt.ToValue(sel.Length())
}

func (Gq) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, node := range sel.Nodes {
			if !yield(node) {
				return
			}
		}
	})
}

type selector struct {
	sel cascadia.Selector
}

type gq struct {
	sel *goquery.Selection
}

var (
	typeSelector  = reflect.TypeOf((*selector)(nil))
	typeSelection = reflect.TypeOf((*gq)(nil))
)

func thisToSel(rt *sobek.Runtime, this sobek.Value) *goquery.Selection {
	if this.ExportType() == typeSelection {
		return this.Export().(*gq).sel
	}
	panic(rt.NewTypeError(`Value must be of type gq.Selection`))
}

// toSelection converts content to goquery.Selection.
// from string, []string, *html.Node, []*html.Node
func toSelection(rt *sobek.Runtime, v sobek.Value) *goquery.Selection {
	switch data := v.Export().(type) {
	default:
		panic(rt.NewTypeError("gq: unexpected type %T", v))
	case nil:
		return new(goquery.Selection)
	case []any:
		nodes := make([]*html.Node, len(data))
		var ok bool
		for i, node := range data {
			nodes[i], ok = node.(*html.Node)
			if !ok {
				panic(rt.NewTypeError("gq: unexpected type %T in array", v))
			}
		}
		return nodesToSel(nodes)
	case *gq:
		return data.sel
	case *goquery.Selection:
		return data
	case *html.Node:
		return goquery.NewDocumentFromNode(data).Selection
	case []*html.Node:
		return nodesToSel(data)
	case []string:
		node, err := htmlutil.Parse(strings.Join(data, ""))
		if err != nil {
			js.Throw(rt, err)
		}
		return goquery.NewDocumentFromNode(node).Selection.Children()
	case fmt.Stringer:
		node, err := htmlutil.Parse(data.String())
		if err != nil {
			js.Throw(rt, err)
		}
		return goquery.NewDocumentFromNode(node).Selection.Children()
	case string:
		node, err := htmlutil.Parse(data)
		if err != nil {
			js.Throw(rt, err)
		}
		return goquery.NewDocumentFromNode(node).Selection.Children()
	}
}

func nodesToSel(nodes []*html.Node) *goquery.Selection {
	root := htmlutil.MergeNode(nodes)
	return goquery.NewDocumentFromNode(root).Children()
}

// compileMatcher compiles the selector string s and returns
// the corresponding Matcher. If s is an invalid selector string,
// it returns a Matcher that fails all matches.
func compileMatcher(s string) goquery.Matcher {
	cs, err := cascadia.Compile(s)
	if err != nil {
		return invalidMatcher{}
	}
	return cs
}

// invalidMatcher is a Matcher that always fails to match.
type invalidMatcher struct{}

func (invalidMatcher) Match(*html.Node) bool            { return false }
func (invalidMatcher) MatchAll(*html.Node) []*html.Node { return nil }
func (invalidMatcher) Filter([]*html.Node) []*html.Node { return nil }

type match struct{}

func (match) Match(*html.Node) bool              { return true }
func (match) MatchAll(n *html.Node) []*html.Node { return []*html.Node{n} }
func (match) Filter(n []*html.Node) []*html.Node { return n }
