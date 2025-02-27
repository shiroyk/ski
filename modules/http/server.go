package http

import (
	"reflect"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

type Server struct{}

func (s *Server) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("listening", rt.ToValue(s.listening), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	_ = p.Set("close", s.append)
	_ = p.Set("closeAllConnections", s.delete)
	_ = p.Set("listen", s.forEach)

	_ = p.SetSymbol(sobek.SymHasInstance, func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue(call.Argument(0).ExportType() == typeServer)
	})
	return p
}

func (s *Server) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	params := call.Argument(0)

	var ret formData
	ret.data = make(map[string][]sobek.Value)

	obj := rt.ToValue(&ret).ToObject(rt)
	_ = obj.SetPrototype(call.This.Prototype())

	if !sobek.IsUndefined(params) {
		callable, ok := sobek.AssertFunction(obj.Get("append"))
		if !ok {
			panic(rt.NewTypeError("invalid formData prototype"))
		}
		if params.ExportType().Kind() != reflect.Map {
			panic(rt.NewTypeError("invalid formData constructor argument"))
		}
		object := params.ToObject(rt)
		for _, key := range object.Keys() {
			_, err := callable(obj, rt.ToValue(key), object.Get(key))
			if err != nil {
				js.Throw(rt, err)
			}
		}
	}

	return obj
}

func (s *Server) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := f.prototype(rt)
	ctor := rt.ToValue(f.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

type server struct {
}

var typeServer = reflect.TypeOf((*server)(nil))

func toServer(rt *sobek.Runtime, value sobek.Value) *server {
	if value.ExportType() == typeServer {
		return value.Export().(*server)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Server`))
}
