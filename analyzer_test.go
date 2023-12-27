package cloudcat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func eval() func(any, string) (any, error) {
	rt := goja.New()
	program := goja.MustCompile("<eval>", "(c, code)=>eval(code)", false)
	callable, err := rt.RunProgram(program)
	if err != nil {
		panic(err)
	}
	call, ok := goja.AssertFunction(callable)
	if !ok {
		panic("err init executor")
	}
	return func(c any, a string) (any, error) {
		value, err := call(goja.Undefined(), rt.ToValue(c), rt.ToValue(a))
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return value.Export(), nil
	}
}

type ap struct {
	eval func(any, string) (any, error)
}

func (p *ap) GetString(_ *plugin.Context, c any, a string) (string, error) {
	v, err := p.eval(c, a)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (p *ap) GetStrings(_ *plugin.Context, c any, a string) ([]string, error) {
	v, err := p.eval(c, a)
	if err != nil {
		return nil, err
	}
	slice, ok := v.([]any)
	if !ok {
		slice = []any{v}
	}
	ret := make([]string, len(slice))
	for i, v := range slice {
		if s, ok := v.(string); ok {
			ret[i] = s
		} else {
			bytes, _ := json.Marshal(v)
			ret[i] = string(bytes)
		}
	}
	return ret, nil
}

func (p *ap) GetElement(_ *plugin.Context, c any, a string) (string, error) {
	return p.GetString(nil, c, a)
}

func (p *ap) GetElements(_ *plugin.Context, c any, a string) ([]string, error) {
	return p.GetStrings(nil, c, a)
}

func TestAnalyzer(t *testing.T) {
	ctx := plugin.NewContext(plugin.ContextOptions{})
	parser.Register("ap", &ap{eval()})
	testCases := []struct {
		schema string
		want   any
	}{
		{
			`
{ ap: '"foo"' }
`, `"foo"`,
		},
		{
			`
- ap: "null"
- or
- ap: '"foo"'
`, `"foo"`,
		},
		{
			`
- ap: "null"
- or
- ap: "null"
- or
- ap: '"foo"'
`, `"foo"`,
		},
		{
			`
- ap: '"foo"'
- and
- ap: '"bar"'
`, `"foobar"`,
		},
		{
			`
- ap: '"foo"'
- and
- ap: '"bar"'
- and
- ap: '"aaa"'
`, `"foobaraaa"`,
		},
		{
			`
type: integer
rule: { ap: '1' }
`, 1,
		},
		{
			`
type: boolean
rule: { ap: '1' }
`, true,
		},
		{
			`
type: number
rule: { ap: '2.1' }
`, 2.1,
		},
		{
			`
type: array
rule:
 - ap: '1'
 - and
 - ap: '2'
`, `["1","2"]`,
		},
		{
			`
type: object
properties:
 string: { ap: '"str"' }
 integer: !integer { ap: '1' }
 number: !number { ap: '1.1' }
 boolean: !boolean { ap: '1' }
 array: !array { ap: "[\"i1\", \"i2\"]" }
 object: !object { ap: "({\"foo\":\"bar\"})" }
`, `{"array":["i1","i2"],"boolean":true,"integer":1,"number":1.1,"object":{"foo":"bar"},"string":"str"}`,
		},
		{
			`
type: object
format: number
rule: { ap: '({"foo":"1.1"})' }
`, `{"foo":1.1}`,
		},
		{
			`
type: array
properties:
 n: !number { ap: '12' }
`, `[{"n":12}]`,
		},
		{
			`
type: array
format: number
rule: { ap: "1" }
`, `[1]`,
		},
		{
			`
type: object
properties:
 ? ap: '"k"'
 : ap: '"v"'
`, `{"k":"v"}`,
		},
		{
			`
type: object
properties:
 $key: { ap: '"k"' }
 $value: { ap: '"v"' }
`, `{"k":"v"}`,
		},
		{
			`
type: object
init: { ap: "[1,2,3]" }
properties:
 ? ap: c
 : ap: c + 1
`, `{"1":"11", "2":"21", "3":"31"}`,
		},
		{
			`
type: object
init: { ap: '["a","b","c",1,2,3]' }
properties:
 $key: { ap: c }
 $value: { ap: c }
`, `{"1":"1", "2":"2", "3":"3", "a":"a", "b":"b", "c":"c"}`,
		},
		{
			`
type: object
properties:
 num: !integer { ap: '2' }
 msg: { ap: '"foooo"' }
 $after: { ap: c.num = c.num + 1; c.msg = "hello"  }
`, `{"num":3,"msg":"hello"}`,
		},
	}
	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			s := new(Schema)
			err := yaml.Unmarshal([]byte(testCase.schema), s)
			if err != nil {
				t.Fatal(err)
			}
			result := Analyze(ctx, s, "")
			if want, ok := testCase.want.(string); ok {
				bytes, err := json.Marshal(result)
				assert.NoError(t, err)
				assert.JSONEq(t, want, string(bytes))
				return
			}
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()
	formatter := new(defaultFormatHandler)
	testCases := []struct {
		data any
		typ  Type
		want any
	}{
		{"", StringType, ""},
		{"1", StringType, "1"},
		{"2.1", NumberType, 2.1},
		{"", NumberType, nil},
		{"3", IntegerType, 3},
		{"1", BooleanType, true},
		{"", BooleanType, nil},
		{`{"k":"v"}`, ObjectType, map[string]any{"k": "v"}},
		{`[1,2]`, ArrayType, []any{1.0, 2.0}},
		{[]string{"1", "2"}, IntegerType, []any{1, 2}},
		{map[string]any{"k": "1"}, IntegerType, map[string]any{"k": 1}},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("Cases %v", i), func(t *testing.T) {
			got, err := formatter.Format(testCase.data, testCase.typ)
			assert.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}

	errCases := []struct {
		data any
		typ  Type
		want any
	}{
		{"9-", IntegerType, nil},
		{"114", BooleanType, nil},
		{[]string{"1", "?"}, IntegerType, nil},
		{map[string]any{"k": "!"}, NumberType, nil},
	}

	for i, testCase := range errCases {
		t.Run(fmt.Sprintf("Err cases %v", i), func(t *testing.T) {
			got, err := formatter.Format(testCase.data, testCase.typ)
			assert.Error(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}
