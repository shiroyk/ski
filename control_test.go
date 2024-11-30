package ski

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type if_int struct{}

func (s if_int) Exec(_ context.Context, arg any) (any, error) {
	_, ok := arg.(int)
	if ok {
		return arg, nil
	}
	return nil, ErrYield
}

func TestControl(t *testing.T) {
	testCases := []struct {
		e    Executor
		arg  any
		want any
	}{
		{_each{if_int{}}, []any{"1", 1, "2", 1, "3", 4}, []any{1, 1, 4}},
		{_each{_if_contains("3")}, []any{"1", 2, "3", 4, "3"}, []any{"3", "3"}},
		{_each{Pipe{_if_contains("3"), _inc{}}}, []any{"1", 2, "3", 4, "3"}, []any{4, 4}},
		{_map{Pipe{_if_contains("3"), _str_prefix("key")}, String("value")}, []any{"1", "2", "3"}, map[string]any{"key3": "value"}},
	}
	for i, c := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			v, err := c.e.Exec(context.Background(), c.arg)
			if assert.NoError(t, err) {
				assert.Equal(t, c.want, v)
			}
		})
	}
}
