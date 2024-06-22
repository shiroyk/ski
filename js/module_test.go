package js

import (
	"context"
	"strconv"
	"testing"

	"github.com/shiroyk/ski"
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
					if s, ok := v.(ski.Iterator); ok {
						for j := 0; j < s.Len(); j++ {
							assert.Equal(t, c.excepted.([]any)[j], s.At(j), "at %d", j)
						}
					} else {
						assert.Equal(t, c.excepted, v)
					}
				}
			}
		})
	}
}
