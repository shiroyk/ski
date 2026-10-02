package common

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type probeDict struct {
	Flag bool
	Name string
	Any  sobek.Value
}

func TestExportDict(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	rt := vm.Runtime()

	export := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		dict := ExportDict[probeDict](rt, call.Argument(0), call.Argument(1).String())
		return rt.ToValue(map[string]any{"flag": dict.Flag, "name": dict.Name})
	}

	result, err := vm.RunModule(context.Background(), `
		export default (exportDict) => {
			const capture = (fn) => {
				try { return fn(); } catch (err) { return err.code + ':' + err.message; }
			};
			return {
				empty: exportDict(undefined, 'options'),
				coerced: exportDict({flag: 'yes', name: 1}, 'options'),
				number: capture(() => exportDict(5, 'options')),
				array: capture(() => exportDict([], 'options')),
				renamed: capture(() => exportDict(5, 'dict')),
			};
		}
	`, export)
	require.NoError(t, err)

	obj := result.ToObject(rt)
	empty := obj.Get("empty").ToObject(rt)
	assert.False(t, empty.Get("flag").ToBoolean())
	assert.Equal(t, "", empty.Get("name").String())

	coerced := obj.Get("coerced").ToObject(rt)
	assert.True(t, coerced.Get("flag").ToBoolean())
	assert.Equal(t, "1", coerced.Get("name").String())

	assert.Contains(t, obj.Get("number").String(), "ERR_INVALID_ARG_TYPE")
	assert.Contains(t, obj.Get("array").String(), "ERR_INVALID_ARG_TYPE")
	assert.Contains(t, obj.Get("renamed").String(), `The "dict" argument must be of type object`)
}
