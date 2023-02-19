package js

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vm := newVM(false, nil)

	testCases := []struct {
		script, want string
		isErr        bool
	}{
		{"return 1", "1", true},
		{"2", "2", false},
		{"a = 1; a + 2", "3", false},
		{"(() => 4)()", "4", false},
		{"[5]", "5", false},
		{"a = {'key':6}; a", "[object Object]", false},
		{"JSON.stringify({'key':7})", `{"key":7}`, false},
		{"JSON.stringify([8])", `[8]`, false},
		{"(async () => 9)()", `[object Promise]`, false},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			v, err := vm.RunString(context.Background(), c.script)
			if err != nil {
				if c.isErr {
					return
				}
				t.Fatal(err)
			}

			assert.Equal(t, c.want, v.String())
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
