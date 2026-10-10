package nodetest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/modules/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessShim(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(js.WithInitial(initShims))
	vm.SetFileLoader(fileLoader)

	value, err := vm.RunString(context.Background(), `
		let received;
		process.once("warning", (warning) => { received = warning; });
		process.emitWarning("boom", "DeprecationWarning", "DEP0005");
		const listener = () => { throw new Error("should be removed"); };
		process.addListener("exit", listener);
		process.removeListener("exit", listener);
		[
			received instanceof CustomEvent,
			received.type,
			received.detail.message,
			received.name,
			received.message,
			received.code,
			process instanceof EventTarget,
			// the dispatch result is whether the event was not canceled
			process.emit("exit"),
			// node only APIs are not part of an EventTarget
			typeof process.removeAllListeners,
		].join(",");
	`)
	require.NoError(t, err)
	assert.Equal(t, "true,warning,boom,DeprecationWarning,boom,DEP0005,true,true,undefined", value.String())
}

// newProcessShim implements the parts of the `process` global that the node
// test suite uses, the events are handled by an events.EventTarget.
func newProcessShim(rt *sobek.Runtime) *sobek.Object {
	target := events.NewEventTarget(rt)
	proc := target.ValueOf()

	once := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		options := rt.NewObject()
		_ = options.Set("once", true)
		add, _ := sobek.AssertFunction(proc.Get("addEventListener"))
		if _, err := add(proc, call.Argument(0), call.Argument(1), options); err != nil {
			js.Throw(rt, err)
		}
		return proc
	}

	// dispatch creates a CustomEvent and dispatches it to the target, node
	// passes the arguments to the listener, so the payload properties are
	// copied to the event to keep the node style listeners working.
	dispatch := func(name string, args ...sobek.Value) bool {
		var detail sobek.Value
		if len(args) > 0 {
			detail = args[0]
		}
		event := events.NewCustomEvent(rt, name, events.CustomEventInit{Detail: detail})
		if payload, ok := detail.(*sobek.Object); ok {
			for _, key := range append([]string{"name", "message", "code", "stack"}, payload.Keys()...) {
				if value := payload.Get(key); value != nil && !sobek.IsUndefined(value) {
					_ = event.ValueOf().Set(key, value)
				}
			}
		}
		return events.DispatchEvent(rt, target, event)
	}

	write := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		data := strings.TrimRight(call.Argument(0).String(), "\n")
		if console, ok := rt.Get("console").(*sobek.Object); ok {
			if log, ok := sobek.AssertFunction(console.Get("log")); ok {
				if _, err := log(sobek.Undefined(), rt.ToValue(data)); err != nil {
					js.Throw(rt, err)
				}
			}
		}
		return rt.ToValue(true)
	}

	emitWarning := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		argument := call.Argument(0)
		var warning *sobek.Object
		if sobek.IsString(argument) {
			err, failed := rt.New(rt.Get("Error"), argument)
			if failed != nil {
				js.Throw(rt, failed)
			}
			warning = err
			if name := call.Argument(1); !sobek.IsUndefined(name) {
				_ = warning.Set("name", name)
			}
			if code := call.Argument(2); !sobek.IsUndefined(code) {
				_ = warning.Set("code", code)
			}
		} else if object, ok := argument.(*sobek.Object); ok {
			warning = object
		} else {
			return sobek.Undefined()
		}
		dispatch("warning", warning)
		return sobek.Undefined()
	}

	nextTick := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		callback, ok := sobek.AssertFunction(call.Argument(0))
		if !ok {
			panic(rt.NewTypeError(`The "callback" argument must be of type function`))
		}
		args := call.Arguments[1:]
		run := func(sobek.FunctionCall, *sobek.Runtime) sobek.Value {
			if _, err := callback(sobek.Undefined(), args...); err != nil {
				js.Throw(rt, err)
			}
			return sobek.Undefined()
		}
		if queue, ok := sobek.AssertFunction(rt.Get("queueMicrotask")); ok {
			if _, err := queue(sobek.Undefined(), rt.ToValue(run)); err != nil {
				js.Throw(rt, err)
			}
			return sobek.Undefined()
		}
		return run(sobek.FunctionCall{}, rt)
	}

	stdout := rt.NewObject()
	_ = stdout.Set("write", write)
	_ = stdout.Set("isTTY", false)
	stderr := rt.NewObject()
	_ = stderr.Set("write", write)
	_ = stderr.Set("isTTY", false)

	_ = proc.Set("env", rt.NewObject())
	_ = proc.Set("argv", rt.NewArray("node"))
	_ = proc.Set("argv0", "node")
	_ = proc.Set("execPath", "node")
	_ = proc.Set("platform", "darwin")
	_ = proc.Set("arch", "arm64")
	_ = proc.Set("pid", 1)
	_ = proc.Set("version", "v26.8.1")
	_ = proc.Set("versions", map[string]any{"node": "26.8.1", "v8": "0.0.0", "modules": "0"})
	_ = proc.Set("stdout", stdout)
	_ = proc.Set("stderr", stderr)
	_ = proc.Set("exitCode", sobek.Undefined())
	_ = proc.Set("on", proc.Get("addEventListener"))
	_ = proc.Set("addListener", proc.Get("addEventListener"))
	_ = proc.Set("once", once)
	_ = proc.Set("removeListener", proc.Get("removeEventListener"))
	_ = proc.Set("emit", func(call sobek.FunctionCall, _ *sobek.Runtime) sobek.Value {
		return rt.ToValue(dispatch(call.Argument(0).String(), call.Arguments[1:]...))
	})
	_ = proc.Set("nextTick", nextTick)
	_ = proc.Set("emitWarning", emitWarning)
	_ = proc.Set("exit", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		code := call.Argument(0).ToInteger()
		_ = proc.Set("exitCode", code)
		panic(rt.NewGoError(fmt.Errorf("process.exit(%d)", code)))
	})
	_ = proc.Set("binding", func(sobek.FunctionCall, *sobek.Runtime) sobek.Value {
		panic(rt.NewTypeError("process.binding is not supported"))
	})

	_ = rt.Set("process", proc)
	return proc
}
