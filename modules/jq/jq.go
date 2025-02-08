// Package jq the json path
package jq

import (
	"reflect"
	"strconv"

	"github.com/grafana/sobek"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("jq", new(Jq))
}

type Jq struct{}

func (Jq) first(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toExpr(rt, call.This).First(doc(rt, call.Argument(0))))
}

func (Jq) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toExpr(rt, call.This).Get(doc(rt, call.Argument(0))))
}

func (Jq) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toExpr(rt, call.This).MustSet(doc(rt, call.Argument(0)), call.Argument(1).Export())
	return sobek.Undefined()
}

func (Jq) setOne(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toExpr(rt, call.This).MustSetOne(doc(rt, call.Argument(0)), call.Argument(1).Export())
	return sobek.Undefined()
}

func (Jq) del(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toExpr(rt, call.This).MustDel(doc(rt, call.Argument(0)))
	return sobek.Undefined()
}

func (Jq) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toExpr(rt, call.This).Has(doc(rt, call.Argument(0))))
}

func (Jq) remove(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toExpr(rt, call.This).MustRemove(doc(rt, call.Argument(0)))
	return sobek.Undefined()
}

func (Jq) removeOne(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	toExpr(rt, call.This).MustRemoveOne(doc(rt, call.Argument(0)))
	return sobek.Undefined()
}

func (j Jq) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("first", j.first)
	_ = p.Set("get", j.get)
	_ = p.Set("set", j.set)
	_ = p.Set("setOne", j.setOne)
	_ = p.Set("del", j.del)
	_ = p.Set("has", j.has)
	_ = p.Set("remove", j.remove)
	_ = p.Set("removeOne", j.removeOne)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("jq") })
	return p
}

func (j Jq) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
		x, err := jp.ParseString(call.Argument(0).String())
		if err != nil {
			js.Throw(rt, err)
		}
		ret := rt.ToValue(&expr{x}).(*sobek.Object)
		_ = ret.SetPrototype(j.prototype(rt))
		return ret
	}), nil
}

type expr struct {
	expr jp.Expr
}

var typeExpr = reflect.TypeOf((*expr)(nil))

func toExpr(rt *sobek.Runtime, this sobek.Value) jp.Expr {
	if this.ExportType() == typeExpr {
		return this.Export().(*expr).expr
	}
	panic(rt.NewTypeError(`Value of "this" must be of type jq.Expr`))
}

func doc(rt *sobek.Runtime, data sobek.Value) any {
	var (
		v   any
		err error
	)
	switch data.ExportType().Kind() {
	default:
		v = toValue(data)
	case reflect.String:
		v, err = oj.ParseString(data.String())
	}
	if err != nil {
		js.Throw(rt, err)
	}
	return v
}

func toValue(v sobek.Value) any {
	if v == nil {
		return nil
	}
	if o, ok := v.(*sobek.Object); ok {
		if o.ClassName() == "Array" {
			return (*indexed)(o)
		}
		return (*keyed)(o)
	}
	return v.Export()
}

func toRaw(v any) any {
	switch t := v.(type) {
	case *keyed:
		return (*sobek.Object)(t)
	case *indexed:
		return (*sobek.Object)(t)
	default:
		return v
	}
}

type keyed sobek.Object

func (k *keyed) ValueForKey(key string) (value any, has bool) {
	v := (*sobek.Object)(k).Get(key)
	if v == nil {
		return nil, false
	}
	return toValue(v), true
}

func (k *keyed) SetValueForKey(key string, value any) {
	_ = (*sobek.Object)(k).Set(key, toRaw(value))
}

func (k *keyed) RemoveValueForKey(key string) {
	_ = (*sobek.Object)(k).Delete(key)
}

func (k *keyed) Keys() []string {
	return (*sobek.Object)(k).Keys()
}

type indexed sobek.Object

func (i *indexed) ValueAtIndex(index int) any {
	return toValue((*sobek.Object)(i).Get(strconv.Itoa(index)))
}

func (i *indexed) SetValueAtIndex(index int, value any) {
	_ = (*sobek.Object)(i).Set(strconv.Itoa(index), toRaw(value))
}

func (i *indexed) Size() int {
	return int((*sobek.Object)(i).Get("length").ToInteger())
}

func (i *indexed) RemoveValueAtIndex(index int) {
	_ = (*sobek.Object)(i).Delete(strconv.Itoa(index))
}
