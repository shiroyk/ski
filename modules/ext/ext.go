package ext

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("ext", new(Ext))
}

type Ext struct{}

func (e Ext) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ext := rt.NewObject()
	_ = ext.Set("context", e.context(rt))
	return ext, nil
}

func (Ext) context(rt *sobek.Runtime) sobek.Value {
	obj := rt.NewObject()
	_ = obj.Set("toString", func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue("[context]")
	})

	proxy := rt.NewProxy(obj, &sobek.ProxyTrapConfig{
		Get: func(target *sobek.Object, property string, receiver sobek.Value) sobek.Value {
			return rt.ToValue(js.Context(rt).Value(property))
		},
		Set: func(target *sobek.Object, property string, value sobek.Value, receiver sobek.Value) bool {
			ctx := js.Context(rt)
			if c, ok := ctx.(interface{ SetValue(key, value any) }); ok {
				c.SetValue(property, value.Export())
				return true
			}
			return false
		},
	})
	return rt.ToValue(proxy).ToObject(rt)
}
