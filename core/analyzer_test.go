package core

import (
	"encoding/json"
	"fmt"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strconv"
	"testing"
)

type analyzerParser struct{}

func (analyzerParser) GetString(_ *plugin.Context, _ any, a string) (string, error) {
	return a, nil
}

func (analyzerParser) GetStrings(_ *plugin.Context, _ any, a string) ([]string, error) {
	return []string{a}, nil
}

func (p analyzerParser) GetElement(ctx *plugin.Context, c any, a string) (string, error) {
	return p.GetString(ctx, c, a)
}

func (p analyzerParser) GetElements(ctx *plugin.Context, c any, a string) ([]string, error) {
	return p.GetStrings(ctx, c, a)
}

func TestAnalyzer(t *testing.T) {
	ctx := plugin.NewContext(plugin.Options{})
	parser.Register("ap", new(analyzerParser))
	testCases := []struct {
		schema string
		want   any
	}{
		{
			`
- ap:
- or
- ap: foo
`, `"foo"`,
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
 string: { ap: 'str' }
 integer:
   type: integer
   rule: { ap: '1' }
 number:
   type: number
   rule: { ap: '1.1' }
 boolean:
   type: boolean
   rule: { ap: '1' }
 array:
   type: string
   format: array
   rule: { ap: "[\"i1\", \"i2\"]" }
 object:
   type: object
   rule: { ap: "{\"foo\":\"bar\"}" }
`, `{"array":["i1","i2"],"boolean":true,"integer":1,"number":1.1,"object":{"foo":"bar"},"string":"str"}`,
		},
		{
			`
type: object
format: number
rule: { ap: "{\"foo\":1.1}" }
`, `{"foo":1.1}`,
		},
		{
			`
type: array
properties:
 n:
   type: number
   rule: { ap: '1' }
`, `[{"n":1}]`,
		},
		{
			`
type: array
format: number
rule: { ap: "1" }
`, `[1]`,
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
		{"1", StringType, "1"},
		{"2.1", NumberType, 2.1},
		{"3", IntegerType, 3},
		{"1", BooleanType, true},
		{`{"k":"v"}`, ObjectType, map[string]any{"k": "v"}},
		{`[1,2]`, ArrayType, []any{1.0, 2.0}},
		{[]string{"1", "2"}, IntegerType, []any{1, 2}},
		{map[string]any{"k": "1"}, IntegerType, map[string]any{"k": 1}},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got, err := formatter.Format(testCase.data, testCase.typ)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, testCase.want, got)
		})
	}
}
