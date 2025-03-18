package buffer

import (
	"encoding/base64"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("node:buffer", modules.Global{
		"Blob":   new(Blob),
		"Buffer": new(Buffer),
		"File":   new(File),
		"atob":   modules.ModuleFunc(atob),
		"btoa":   modules.ModuleFunc(btoa),
	})
}

// atob decodes a base64 string to a string
func atob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	str := call.Argument(0).String()
	bytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(string(bytes))
}

// btoa encodes a string to base64 string
func btoa(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	str := call.Argument(0).String()
	return rt.ToValue(base64.StdEncoding.EncodeToString([]byte(str)))
}
