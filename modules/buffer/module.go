package buffer

import (
	"encoding/base64"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("node:buffer", new(Module))
}

type Module struct{}

func (Module) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.NewObject()
	blob, _ := new(Blob).Instantiate(rt)
	_ = ret.Set("Blob", blob)
	file, _ := new(File).Instantiate(rt)
	blobProto := blob.(*sobek.Object).Prototype()
	fileProto := file.(*sobek.Object).Prototype()
	_ = fileProto.SetPrototype(blobProto)
	_ = fileProto.Set("prototype", blobProto)
	_ = ret.Set("File", file)
	_ = ret.Set("atob", atob)
	_ = ret.Set("btoa", btoa)
	return ret, nil
}

func (Module) Global() {}

func atob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	str := call.Argument(0).String()
	bytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(string(bytes))
}

func btoa(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	str := call.Argument(0).String()
	return rt.ToValue(base64.StdEncoding.EncodeToString([]byte(str)))
}
