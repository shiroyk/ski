package gq

import (
	"reflect"

	"github.com/PuerkitoBio/goquery"
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	htmlutil "github.com/shiroyk/ski/modules/html"
	"golang.org/x/net/html"
)

// eq reduces the set of matched elements to the one at the specified index
func (Gq) eq(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	idx := int(call.Argument(0).ToInteger())
	ret := rt.ToValue(&gq{sel.Eq(idx)}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// filter reduces the set of matched elements to those that match the selector or pass the function's test
func (Gq) filter(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("filter requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)
	prototype := call.This.ToObject(rt).Prototype()

	switch v.ExportType() {
	case typeSelector:
		sel = sel.FilterMatcher(v.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		sel = sel.FilterSelection(nodesToSel(v.Export().([]*html.Node)))
	default:
		if v.ExportType().Kind() == reflect.String {
			sel = sel.Filter(v.String())
		} else {
			callback, ok := sobek.AssertFunction(v)
			if !ok {
				panic(rt.NewTypeError("filter argument not a function"))
			}
			sel = sel.FilterFunction(func(i int, s *goquery.Selection) bool {
				value := rt.ToValue(&gq{s}).(*sobek.Object)
				_ = value.SetPrototype(prototype)
				ret, err := callback(value, rt.ToValue(i), value)
				if err != nil {
					js.Throw(rt, err)
				}
				return ret.ToBoolean()
			})
		}
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(prototype)
	return ret
}

// first reduces the set of matched elements to the first in the set
func (Gq) first(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	ret := rt.ToValue(sel.First()).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// last reduces the set of matched elements to the final one in the set
func (Gq) last(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	ret := rt.ToValue(&gq{sel.Last()}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// has reduces the set of matched elements to those that have a descendant that matches the selector
func (Gq) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("has requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)

	switch v.ExportType() {
	case typeSelector:
		sel = sel.HasMatcher(v.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		sel = sel.HasNodes(v.Export().([]*html.Node)...)
	default:
		sel = sel.Has(v.String())
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// is checks the current matched set of elements against a selector
func (Gq) is(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("is requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)
	var result bool

	switch v.ExportType() {
	case typeSelector:
		result = sel.IsMatcher(v.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		result = sel.IsNodes(v.Export().([]*html.Node)...)
	default:
		result = sel.Is(v.String())
	}

	return rt.ToValue(result)
}

// even reduces the set of matched elements to the even ones in the set
func (Gq) even(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	sel = sel.FilterFunction(func(i int, _ *goquery.Selection) bool {
		return i%2 == 0
	})
	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// add adds elements to the set of matched elements
func (Gq) add(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("add requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)

	switch v.ExportType() {
	case htmlutil.TypeNodes:
		newSel := nodesToSel(v.Export().([]*html.Node))
		sel = sel.AddSelection(newSel)
	default:
		sel = sel.Add(v.String())
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// not removes elements from the set of matched elements
func (Gq) not(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("not requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)

	switch v.ExportType() {
	case typeSelector:
		sel = sel.NotMatcher(v.Export().(*selector).sel)
	default:
		sel = sel.Not(v.String())
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// odd reduces the set of matched elements to the odd ones in the set
func (Gq) odd(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	sel = sel.FilterFunction(func(i int, _ *goquery.Selection) bool {
		return i%2 == 1
	})
	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// slice reduces the set of matched elements to a subset specified by a range of indices
func (Gq) slice(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("slice requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	start := int(call.Argument(0).ToInteger())
	var end int
	if len(call.Arguments) > 1 {
		end = int(call.Argument(1).ToInteger())
	} else {
		end = len(sel.Nodes)
	}

	ret := rt.ToValue(&gq{sel.Slice(start, end)}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// map passes each element in the current matched set through a function
func (Gq) map_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("map requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("map argument not a function"))
	}
	var results []any
	prototype := call.This.ToObject(rt).Prototype()

	for i, s := range sel.EachIter() {
		value := rt.ToValue(&gq{s}).(*sobek.Object)
		_ = value.SetPrototype(prototype)
		ret, err := callback(value, rt.ToValue(i), value)
		if err != nil {
			js.Throw(rt, err)
		}
		results = append(results, ret.Export())
	}

	return rt.ToValue(results)
}

// each executing a function for each matched element.
func (Gq) each(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("each requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("each argument not a function"))
	}
	prototype := call.This.ToObject(rt).Prototype()

	for i, s := range sel.EachIter() {
		value := rt.ToValue(&gq{s}).(*sobek.Object)
		_ = value.SetPrototype(prototype)
		_, err := callback(value, rt.ToValue(i), value)
		if err != nil {
			js.Throw(rt, err)
		}
	}

	return call.This
}
