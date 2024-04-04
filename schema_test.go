package ski

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"testing"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type _inc struct{}

func (_inc) Exec(_ context.Context, v any) (ret any, err error) {
	return cast.ToInt(v) + 1, nil
}

type _dec struct{}

func (_dec) Exec(_ context.Context, v any) (ret any, err error) {
	return cast.ToInt(v) - 1, nil
}

func TestExecutor(t *testing.T) {
	ctx := context.WithValue(context.Background(), "foo", "bar")
	testCases := []struct {
		e    Executor
		arg  any
		want any
	}{
		{_raw{nil}, 1, nil},
		{_map{_raw{"k"}, _raw{nil}}, 1, map[string]any{"k": nil}},
		{_pipe{_inc{}, _inc{}, _dec{}}, 1, 2},
		{KindInt, "1", int32(1)},
		{_or{_raw{nil}, _raw{"b"}}, nil, "b"},
		{_map{_raw{"k"}, _inc{}}, 0, map[string]any{"k": 1}},
		{_each{_inc{}}, []string{"1", "2", "3"}, []any{2, 3, 4}},
		{_map{_raw{"k"}, _inc{}}, []any{1}, map[string]any{"k": 2}},
		{_string_join(""), []string{"1", "2", "3"}, "123"},
		{_pipe{_each{KindString}, _string_join("")}, []any{1, 2, 3}, "123"},
		{_pipe{_each{_inc{}}, _each{_inc{}}}, []any{1, 2, 3}, []any{3, 4, 5}},
		{_each{_map{_raw{"k"}, _inc{}}}, []any{1}, []any{map[string]any{"k": 2}}},
		{_map{_raw{"k"}, _json_parse{}}, `{"foo": "bar"}`, map[string]any{"k": map[string]any{"foo": "bar"}}},
	}
	for i, c := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			v, err := c.e.Exec(ctx, c.arg)
			if assert.NoError(t, err) {
				assert.Equal(t, c.want, v)
			}
		})
	}
}

func TestDebug(t *testing.T) {
	data := new(bytes.Buffer)
	ctx := WithLogger(context.Background(), slog.New(slog.NewTextHandler(data, &slog.HandlerOptions{Level: slog.LevelDebug})))
	v, err := _pipe{_debug("before"), _inc{}, _debug("after")}.Exec(ctx, 1)
	if assert.NoError(t, err) {
		assert.Equal(t, v, 2)
	}
	assert.Regexp(t, "msg=before value=1 | msg=after value=2", data.String())
}

type meta struct {
	exec         Executor
	line, column int
}

func (m *meta) Exec(ctx context.Context, arg any) (any, error) {
	v, err := m.exec.Exec(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("line %d column %d: %s", m.line, m.column, err)
	}
	return v, nil
}

type errexec struct{}

func new_errexec(_ ...Executor) (Executor, error) { return new(errexec), nil }

func (errexec) Exec(context.Context, any) (any, error) {
	return nil, fmt.Errorf("some error")
}

func TestWithMetaWrap(t *testing.T) {
	Register("error", new_errexec)
	v, err := Compile(`$error: ...`,
		WithMeta(func(node *yaml.Node, exec Executor, isParser bool) Executor {
			return &meta{exec, node.Line, node.Column}
		}))
	if assert.NoError(t, err) {
		_, err = v.Exec(context.Background(), ``)
		assert.ErrorContains(t, err, "line 1 column 1: some error")
	}
}

// deepEqual recursively checks if the values (dereferencing pointers) are the same.
func deepEqual(x, y any) bool {
	v1, v2 := reflect.ValueOf(x), reflect.ValueOf(y)
	if v1.Kind() == reflect.Ptr && v2.Kind() == reflect.Ptr {
		return deepEqual(v1.Elem().Interface(), v2.Elem().Interface())
	}
	if v1.Kind() == reflect.Ptr {
		v1 = v1.Elem()
		x = v1.Interface()
	}
	if v2.Kind() == reflect.Ptr {
		v2 = v2.Elem()
		y = v2.Interface()
	}
	if v1.Kind() == reflect.Slice && v2.Kind() == reflect.Slice {
		if v1.Len() != v2.Len() {
			return false
		}
		for i := 0; i < v1.Len(); i++ {
			if !deepEqual(v1.Index(i).Interface(), v2.Index(i).Interface()) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(x, y)
}

func TestCompileBuildIn(t *testing.T) {
	t.Parallel()
	t.Run("top pipe 1", func(t *testing.T) {
		expect := _pipe{
			_map{String("title"), _debug("text")},
			_or{String("title"), _debug("text")}}
		exec, err := Compile(`
$map: &alias
    title:
      $debug: text
$or: *alias`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})

	t.Run("top pipe 2", func(t *testing.T) {
		expect := _pipe{
			_map{String("title"), _debug("text")},
			_map{String("title"), _debug("text")}}
		exec, err := Compile(`
- $map: &alias
    title:
      $debug: text
- $map: *alias`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})

	t.Run("mapping pipe", func(t *testing.T) {
		expect := _map{String("size"), _pipe{_debug("the size"), KindInt64}}
		exec, err := Compile(`
$map:
  size:
    $debug: the size
    $kind: int64`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})

	t.Run("sequence pipe", func(t *testing.T) {
		expect := _map{String("size"), _pipe{_debug("the size"), KindInt64}}
		exec, err := Compile(`
$map:
  size:
    - $debug: the size
    - $kind: int64`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})

	t.Run("pipe", func(t *testing.T) {
		expect := _map{String("size"), _pipe{_debug("the size"), KindInt}}
		exec, err := Compile(`
$map:
  size:
    $pipe:
      - $debug: the size
      - $kind: int32`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})
}
