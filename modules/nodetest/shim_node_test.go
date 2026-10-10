package nodetest

import (
	"slices"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// newNodeTestShim implements the describe()/it()/test() subset of node:test,
// the tests are executed immediately like the other suites do.
func newNodeTestShim(rt *sobek.Runtime) *sobek.Object {
	run := func(call sobek.FunctionCall) sobek.Value {
		for i := range slices.Backward(call.Arguments) {
			if function, ok := sobek.AssertFunction(call.Argument(i)); ok {
				if _, err := function(sobek.Undefined()); err != nil {
					js.Throw(rt, err)
				}
				break
			}
		}
		return sobek.Undefined()
	}
	noop := func(sobek.FunctionCall) sobek.Value { return sobek.Undefined() }

	module := rt.NewObject()
	_ = module.Set("describe", run)
	_ = module.Set("it", run)
	_ = module.Set("test", run)
	_ = module.Set("before", noop)
	_ = module.Set("after", noop)
	_ = module.Set("beforeEach", noop)
	_ = module.Set("afterEach", noop)
	return module
}
