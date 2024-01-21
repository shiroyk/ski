package ski

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
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
		{_stringJoin(""), []string{"1", "2", "3"}, "123"},
		{_pipe{_each{KindString}, _stringJoin("")}, []any{1, 2, 3}, "123"},
		{_pipe{_each{_inc{}}, _each{_inc{}}}, []any{1, 2, 3}, []any{3, 4, 5}},
		{_each{_map{_raw{"k"}, _inc{}}}, []any{1}, []any{map[string]any{"k": 2}}},
		{_map{_raw{"k"}, _jsonParse{}}, `{"foo": "bar"}`, map[string]any{"k": map[string]any{"foo": "bar"}}},
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

func (errexec) Exec(context.Context, any) (any, error) {
	return nil, fmt.Errorf("some error")
}

func TestWithMetaWrap(t *testing.T) {
	v, err := Compile(`$error: ...`,
		WithMeta(func(node *yaml.Node, exec Executor, isParser bool) Executor {
			return &meta{exec, node.Line, node.Column}
		}),
		WithExecutorMap(ExecutorMap{"error": func(args ...Executor) (Executor, error) {
			return errexec{}, nil
		}}))
	if assert.NoError(t, err) {
		_, err = v.Exec(context.Background(), ``)
		assert.ErrorContains(t, err, "line 1 column 1: some error")
	}
}

type p struct{}

func (p) Value(string) (Executor, error)    { return String("p.value"), nil }
func (p) Element(string) (Executor, error)  { return String("p.element"), nil }
func (p) Elements(string) (Executor, error) { return String("p.elements"), nil }

func TestCompile(t *testing.T) {
	Register("p", p{})
	testCases := []struct {
		s string
		e Executor
	}{
		{`$p: foo`, String("p.value")},
		{`$p.value: foo`, String("p.value")},
		{`
- $map: &alias
    title:
      $p: text
- $map: *alias`,
			_pipe{
				_map{String("title"), String("p.value")},
				_map{String("title"), String("p.value")}}},
		{`
$map:
  size:
    $p: text
    $debug: the size
    $kind: int64`, _map{String("size"),
			_pipe{String("p.value"), _debug("the size"), KindInt64}}},
		{`
$map:
  size:
    - $p: text
    - $debug: the size
    - $kind: int64`, _map{String("size"),
			_pipe{String("p.value"), _debug("the size"), KindInt64}}},
		{`
$map:
  size:
    $pipe:
      - $p: text
      - $debug: the size
      - $kind: int32`, _map{String("size"),
			_pipe{String("p.value"), _debug("the size"), KindInt}}},
	}
	for i, c := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			v, err := Compile(c.s)
			if assert.NoError(t, err) {
				assert.Equal(t, c.e, v)
			}
		})
	}
}
