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
		{Pipe{_inc{}, _inc{}, _dec{}}, 1, 2},
		{KindInt, "1", int32(1)},
		{_or{_raw{nil}, _raw{"b"}}, nil, "b"},
		{_map{_raw{"k"}, _inc{}}, 0, map[string]any{"k": 1}},
		{_each{_inc{}}, []any{"1", "2", "3"}, []any{2, 3, 4}},
		{_map{_raw{"k"}, _inc{}}, []any{1}, map[string]any{"k": 2}},
		{_str_join(""), []any{"1", "2", "3"}, "123"},
		{_str_split(""), "123", []string{"1", "2", "3"}},
		{Pipe{_each{KindString}, _str_join("")}, []any{1, 2, 3}, "123"},
		{Pipe{_each{_inc{}}, _each{_inc{}}}, []any{1, 2, 3}, []any{3, 4, 5}},
		{_each{_map{_raw{"k"}, _inc{}}}, []any{1}, []any{map[string]any{"k": 2}}},
		{_map{_raw{"k"}, _json_parse{}}, `{"foo": "bar"}`, map[string]any{"k": map[string]any{"foo": "bar"}}},
		{_list_of{String(`1`), String(`2`), String(`3`)}, nil, []any{"1", "2", "3"}},
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
	v, err := Pipe{_debug("before"), _inc{}, _debug("after")}.Exec(ctx, 1)
	if assert.NoError(t, err) {
		assert.Equal(t, v, 2)
	}
	assert.Regexp(t, "msg=before value=1 | msg=after value=2", data.String())
}

type errexec struct {
	line, column int
}

func new_errexec(_ Arguments) (Executor, error) { return new(errexec), nil }

func (e *errexec) Meta(k, v *yaml.Node) error {
	e.line = k.Line
	e.column = k.Column
	return nil
}

func (e errexec) Exec(context.Context, any) (any, error) {
	return nil, fmt.Errorf("line %d column %d: some error", e.line, e.column)
}

func TestMeta(t *testing.T) {
	Register("error", new_errexec)
	v, err := Compile(`$error: ...`)
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
	t.Run("mapping pipe", func(t *testing.T) {
		expect := _map{String("size"), Pipe{_debug("the size"), KindInt64}}
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
		expect := _map{String("size"), Pipe{_debug("the size"), KindInt64}}
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
		expect := _map{String("size"), Pipe{_debug("the size"), KindInt}}
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

	t.Run("mix executor", func(t *testing.T) {
		_, err := Compile(`
$raw:
title: `)
		assert.ErrorContains(t, err, "mix executor and map key")
		_, err = Compile(`
title:
$raw: `)
		assert.ErrorContains(t, err, "mix executor and map key")
	})
}

type testPtr string

func (a *testPtr) Exec(context.Context, any) (any, error) { return string(*a), nil }

func TestCompileAlias(t *testing.T) {
	t.Parallel()
	t.Run("alias reuse", func(t *testing.T) {
		Register("test.ptr", func(arg Arguments) (Executor, error) {
			a := testPtr(arg.GetString(0))
			return &a, nil
		})
		exec, err := Compile(`
$each: &alias
  $test.ptr: 1
$each: *alias`)
		if assert.NoError(t, err) {
			slice := exec.(Pipe)
			if assert.Equal(t, 2, len(slice)) {
				p1, p2 := slice[0].(_each).Executor, slice[1].(_each).Executor
				assert.True(t, p1 == p2, "not reuse executor")
			}
		}
	})

	t.Run("alias sequence merge", func(t *testing.T) {
		expect := Pipe{
			_map{String("title"), Pipe{String("1"), String("2")}},
			_map{String("title"), Pipe{String("1"), String("2")}, String("value"), Pipe{String("1"), String("2"), KindInt64}}}
		exec, err := Compile(`
$map: 
  title: &alias
    - &t1 $raw: 1
    - &t2 $raw: 2
$map:
  title: 
    <<: *alias
  value:
    <<: [ *t1, *t2 ]
    $kind: int64`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})

	t.Run("alias mapping merge", func(t *testing.T) {
		expect := Pipe{
			_map{String("title"), Pipe{String("1"), _debug("text")}},
			_map{String("title"), Pipe{String("1"), _debug("text")}, String("value"), Pipe{String("1"), KindInt64}}}
		exec, err := Compile(`
$map: &alias
  title:
    $raw: &t1 1
    $debug: text
$map:
  <<: *alias
  value:
    <<: *t1
    $kind: int64`)
		if assert.NoError(t, err) {
			assert.True(t, deepEqual(expect, exec))
		}
	})
}
