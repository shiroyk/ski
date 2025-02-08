// Package xpath the xpath executor
package xpath

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/antchfx/xpath"
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
	htmlutil "github.com/shiroyk/ski/modules/html"
	"golang.org/x/net/html"
)

func init() {
	modules.Register("xpath", new(Xpath))
}

type Xpath struct{}

func (x Xpath) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
		ex, err := xpath.Compile(call.Argument(0).String())
		if err != nil {
			js.Throw(rt, err)
		}
		ret := rt.ToValue(&expr{ex}).(*sobek.Object)
		_ = ret.SetPrototype(x.prototype(rt))
		return ret
	}).ToObject(rt)
	_ = ret.Set("innerText", x.innerText)
	return ret, nil
}

func (x Xpath) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("innerText", x.innerText)
	_ = p.Set("querySelector", x.querySelector)
	_ = p.Set("querySelectorAll", x.querySelectorAll)
	return p
}

func nodesText(nodes []*html.Node) []string {
	ret := make([]string, len(nodes))
	for i, n := range nodes {
		ret[i] = htmlquery.InnerText(n)
	}
	return ret
}

func (Xpath) innerText(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := call.This
	switch this.ExportType() {
	case typeExpr:
		nodes := htmlquery.QuerySelectorAll(htmlNode(rt, call.Argument(0)), toExpr(rt, call.This))
		return rt.ToValue(nodesText(nodes))
	case htmlutil.TypeNodes:
		return rt.ToValue(nodesText(this.Export().([]*html.Node)))
	case htmlutil.TypeNode:
		return rt.ToValue(htmlquery.InnerText(this.Export().(*html.Node)))
	}

	switch t := call.Argument(0).Export().(type) {
	case []*html.Node:
		return rt.ToValue(nodesText(t))
	case *html.Node:
		return rt.ToValue(htmlquery.InnerText(t))
	default:
		return sobek.Null()
	}
}

func (x Xpath) querySelector(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	node := htmlquery.QuerySelector(htmlNode(rt, call.Argument(0)), toExpr(rt, call.This))
	if node == nil {
		return sobek.Null()
	}
	ret := rt.ToValue(node).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

func (x Xpath) querySelectorAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	nodes := htmlquery.QuerySelectorAll(htmlNode(rt, call.Argument(0)), toExpr(rt, call.This))
	if len(nodes) == 0 {
		return sobek.Null()
	}
	ret := rt.ToValue(nodes).(*sobek.Object)
	_ = ret.Set("innerText", x.innerText)
	return ret
}

type expr struct {
	expr *xpath.Expr
}

var typeExpr = reflect.TypeOf((*expr)(nil))

func toExpr(rt *sobek.Runtime, this sobek.Value) *xpath.Expr {
	if this.ExportType() == typeExpr {
		return this.Export().(*expr).expr
	}
	panic(rt.NewTypeError(`Value of "this" must be of type xpath.Expr`))
}

func htmlNode(rt *sobek.Runtime, v sobek.Value) *html.Node {
	switch data := v.Export().(type) {
	default:
		panic(rt.NewTypeError("unexpected type %T", data))
	case nil:
		return &html.Node{Type: html.DocumentNode}
	case []any:
		nodes := make([]*html.Node, len(data))
		var ok bool
		for i, node := range data {
			nodes[i], ok = node.(*html.Node)
			if !ok {
				panic(rt.NewTypeError("xpath: unexpected type %T in array", v))
			}
		}
		return htmlutil.MergeNode(nodes)
	case []*html.Node:
		return htmlutil.MergeNode(data)
	case *html.Node:
		return data
	case []string:
		node, err := htmlutil.Parse(strings.Join(data, ""))
		if err != nil {
			js.Throw(rt, err)
		}
		return node
	case fmt.Stringer:
		node, err := htmlutil.Parse(data.String())
		if err != nil {
			js.Throw(rt, err)
		}
		return node
	case string:
		node, err := htmlutil.Parse(data)
		if err != nil {
			js.Throw(rt, err)
		}
		return node
	}
}
