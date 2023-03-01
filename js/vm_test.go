package js

import (
	"context"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat/js/common"
	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vm := newVM(false, nil)

	testCases := []struct {
		script string
		want   any
	}{
		{"2", 2},
		{"a = 1; a + 2", 3},
		{"(() => 4)()", 4},
		{"[5]", []any{int64(5)}},
		{"a = {'key':'foo'}; a", map[string]any{"key": "foo"}},
		{"JSON.stringify({'key':7})", `{"key":7}`},
		{"JSON.stringify([8])", `[8]`},
		{"(async () => 9)()", 9},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			v, err := vm.RunString(context.Background(), c.script)
			assert.NoError(t, err)
			vv, err := common.Unwrap(v)
			assert.NoError(t, err)
			assert.EqualValues(t, c.want, vv)
		})
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := newVM(false, nil).RunString(ctx, `while(true){}`)
	assert.ErrorContains(t, err, context.DeadlineExceeded.Error())
}

func TestUseStrict(t *testing.T) {
	t.Parallel()
	vm := newVM(true, nil)
	_, err := vm.RunString(context.Background(), `eval('a = 1');a`)
	if err != nil {
		assert.Contains(t, err.Error(), "ReferenceError: a is not defined")
	}
}
