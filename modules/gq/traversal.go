package gq

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/grafana/sobek"
	htmlutil "github.com/shiroyk/ski/modules/html"
	"golang.org/x/net/html"
)

// find gets the descendants of each element in the current set of matched elements
func (Gq) find(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("find requires at least 1 argument"))
	}
	v := call.Argument(0)
	switch v.ExportType() {
	case typeSelector:
		sel = sel.FindMatcher(v.Export().(*selector).sel)
	default:
		sel = sel.Find(v.String())
	}
	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// children gets the children of each element in the set of matched elements
func (Gq) children(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("children requires at least 1 argument"))
	}
	v := call.Argument(0)
	switch v.ExportType() {
	case typeSelector:
		sel = sel.ChildrenMatcher(v.Export().(*selector).sel)
	default:
		sel = sel.ChildrenFiltered(v.String())
	}
	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// parent gets the parent of each element in the current set of matched elements
func (Gq) parent(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.ParentMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.ParentFiltered(v.String())
		}
	} else {
		sel = sel.Parent()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// parents gets the ancestors of each element in the current set of matched elements
func (Gq) parents(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.ParentsMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.ParentsFiltered(v.String())
		}
	} else {
		sel = sel.Parents()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// next gets the immediately following sibling
func (Gq) next(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.NextMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.NextFiltered(v.String())
		}
	} else {
		sel = sel.Next()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// prev gets the immediately preceding sibling
func (Gq) prev(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.PrevMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.PrevFiltered(v.String())
		}
	} else {
		sel = sel.Prev()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// siblings gets the siblings of each element
func (Gq) siblings(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.SiblingsMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.SiblingsFiltered(v.String())
		}
	} else {
		sel = sel.Siblings()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// nextAll gets all following siblings of each element
func (Gq) nextAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.NextAllMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.NextAllFiltered(v.String())
		}
	} else {
		sel = sel.NextAll()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// prevAll gets all preceding siblings of each element
func (Gq) prevAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)

	if len(call.Arguments) > 0 {
		v := call.Argument(0)
		switch v.ExportType() {
		case typeSelector:
			sel = sel.PrevAllMatcher(v.Export().(*selector).sel)
		default:
			sel = sel.PrevAllFiltered(v.String())
		}
	} else {
		sel = sel.PrevAll()
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

func toFilter(v sobek.Value) goquery.Matcher {
	if !sobek.IsUndefined(v) {
		switch v.ExportType() {
		case typeSelector:
			return v.Export().(*selector).sel
		default:
			return compileMatcher(v.String())
		}
	}
	return match{}
}

// nextUntil gets all following siblings up to but not including the element matched by the selector
func (Gq) nextUntil(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("nextUntil requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	until := call.Argument(0)
	filter := toFilter(call.Argument(1))

	switch until.ExportType() {
	case typeSelector:
		sel = sel.NextFilteredUntilMatcher(filter, until.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		sel = sel.NextMatcherUntilNodes(until.Export().(*selector).sel, until.Export().([]*html.Node)...)
	default:
		sel = sel.NextFilteredUntilMatcher(filter, compileMatcher(until.String()))
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// prevUntil gets all preceding siblings up to but not including the element matched by the selector
func (Gq) prevUntil(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("prevUntil requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	until := call.Argument(0)
	filter := toFilter(call.Argument(1))

	switch until.ExportType() {
	case typeSelector:
		sel = sel.PrevFilteredUntilMatcher(filter, until.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		sel = sel.PrevMatcherUntilNodes(until.Export().(*selector).sel, until.Export().([]*html.Node)...)
	default:
		sel = sel.PrevFilteredUntilMatcher(filter, compileMatcher(until.String()))
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// parentsUntil gets the ancestors up to but not including the element matched by the selector
func (Gq) parentsUntil(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("parentsUntil requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	until := call.Argument(0)
	filter := toFilter(call.Argument(1))

	switch until.ExportType() {
	case typeSelector:
		sel = sel.ParentsFilteredUntilMatcher(filter, until.Export().(*selector).sel)
	case htmlutil.TypeNodes:
		sel = sel.ParentsMatcherUntilNodes(until.Export().(*selector).sel, until.Export().([]*html.Node)...)
	default:
		sel = sel.ParentsFilteredUntilMatcher(filter, compileMatcher(until.String()))
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// closest gets the first element that matches the selector by testing the element itself and traversing up
func (Gq) closest(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("closest requires at least 1 argument"))
	}

	sel := thisToSel(rt, call.This)
	v := call.Argument(0)

	switch v.ExportType() {
	case typeSelector:
		sel = sel.ClosestMatcher(v.Export().(*selector).sel)
	default:
		sel = sel.Closest(v.String())
	}

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}

// contents gets the children including text and comment nodes
func (Gq) contents(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	sel := thisToSel(rt, call.This)
	sel = sel.Contents()

	ret := rt.ToValue(&gq{sel}).(*sobek.Object)
	_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
	return ret
}
