package assert

import (
	"errors"
	"fmt"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("assert", new(Assert))
}

type Assert struct{}

func (a Assert) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.ToValue(a.true).ToObject(rt)
	_ = ret.Set("true", a.true)
	_ = ret.Set("equal", a.equal)
	return ret, nil
}

func (Assert) true(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.Argument(0).ToBoolean() {
		return sobek.Undefined()
	}

	var message string
	if msg := call.Argument(1); !sobek.IsUndefined(msg) {
		message = msg.String()
	} else {
		message = `Expected true but got false`
	}

	panic(rt.NewGoError(errors.New(message)))

	return sobek.Undefined()
}

func (Assert) equal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	a, b := call.Argument(0), call.Argument(1)

	if a.Equals(b) {
		return sobek.Undefined()
	}

	var message string
	if msg := call.Argument(2); !sobek.IsUndefined(msg) {
		var args []sobek.Value
		if len(call.Arguments) > 3 {
			args = call.Arguments[3:]
		}
		message = js.Format(rt, args...)
	} else {
		message = fmt.Sprintf(`Expected equal but got %s  %s`, a.String(), b.String())
	}

	panic(rt.NewGoError(errors.New(message)))

	return sobek.Undefined()
}
