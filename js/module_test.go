package js

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutor(t *testing.T) {
	t.Parallel()
	vm := NewVM(WithModuleLoader(NewModuleLoader()))

	cases := []struct {
		script   string
		excepted any
	}{
		{`export default () => 1`, int64(1)},
		{`export default () => [1]`, []any{int64(1)}},
		{`export default () => ["a"]`, []any{"a"}},
		{`export default () => [{"a": 1}]`, []any{map[string]any{"a": int64(1)}}},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			module, err := vm.Loader().CompileModule("", c.script)
			if assert.NoError(t, err) {
				exec := Executor{module}
				v, err := exec.Exec(context.Background(), nil)
				if assert.NoError(t, err) {
					if s, ok := v.([]any); ok {
						for j := 0; j < len(s); j++ {
							assert.Equal(t, c.excepted.([]any)[j], s[j], "at %d", j)
						}
					} else {
						assert.Equal(t, c.excepted, v)
					}
				}
			}
		})
	}
}

func TestExecutorArgument(t *testing.T) {
	t.Parallel()
	vm := NewVM(WithModuleLoader(NewModuleLoader()))
	module, err := vm.Loader().CompileModule("", `module.exports = (ori) => ori`)
	if assert.NoError(t, err) {
		exec := Executor{module}
		v, err := exec.Exec(context.Background(), "ori")
		if assert.NoError(t, err) {
			assert.Equal(t, v, "ori")
		}
	}
}
