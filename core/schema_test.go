package cloudcat

import (
	"strconv"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSchemaYaml(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Yaml   string
		Schema *Schema
	}{
		{
			`
{ p: foo }`, NewSchema(StringType).
				SetRule(NewSteps("p", "foo")),
		},
		{
			`
- p: foo
  p: bar`, NewSchema(StringType).
				SetRule(NewSteps("p", "foo", "p", "bar")),
		},
		{
			`
- p: foo
- p: bar
- p: title`, NewSchema(StringType).
				SetRule(NewSteps("p", "foo", "p", "bar", "p", "title")),
		},
		{
			`
- p: foo
- or
- p: title`, NewSchema(StringType).
				SetRule(NewOr(NewSteps("p", "foo"), NewSteps("p", "title"))),
		},
		{
			`
- p: foo
- and
- p: bar
- or
- p: body`, NewSchema(StringType).
				SetRule(NewOr(
					NewAnd(NewSteps("p", "foo"), NewSteps("p", "bar")),
					NewSteps("p", "body"),
				)),
		},
		{
			`
- - p: foo
  - p: bar
- or
- - p: title
  - p: body`, NewSchema(StringType).
				SetRule(NewOr(NewSteps("p", "foo", "p", "bar"), NewSteps("p", "title", "p", "body"))),
		},
		{
			`
type: integer
rule: { p: foo }`, NewSchema(IntegerType).
				SetRule(NewSteps("p", "foo")),
		},
		{
			`
type: number
rule: { p: foo }`, NewSchema(NumberType).
				SetRule(NewSteps("p", "foo")),
		},
		{
			`
type: boolean
rule: { p: foo }`, NewSchema(BooleanType).
				SetRule(NewSteps("p", "foo")),
		},
		{
			`
type: object
properties:
  context:
    type: string
    format: boolean
    rule: { p: foo }`, NewSchema(ObjectType).
				AddProperty("context", *NewSchema(StringType, BooleanType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
properties:
  context: !string/boolean { p: foo }`, NewSchema(ObjectType).
				AddProperty("context", *NewSchema(StringType, BooleanType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: array
init: { p: foo }
properties:
  context:
    type: string
    format: integer
    rule: { p: foo }`, NewSchema(ArrayType).
				SetInit(NewSteps("p", "foo")).
				AddProperty("context", *NewSchema(StringType, IntegerType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
init: { p: foo }
properties:
  context:
    type: number
    rule: { p: foo }`, NewSchema(ObjectType).
				SetInit(NewSteps("p", "foo")).
				AddProperty("context", *NewSchema(NumberType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
init: { p: foo }
properties:
  context: !number { p: foo }`, NewSchema(ObjectType).
				SetInit(NewSteps("p", "foo")).
				AddProperty("context", *NewSchema(NumberType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
init:
  - p: foo
  - or
  - p: bar
properties:
  context:
    type: number
    rule: { p: foo }`, NewSchema(ObjectType).
				SetInit(NewOr(NewSteps("p", "foo"), NewSteps("p", "bar"))).
				AddProperty("context", *NewSchema(NumberType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
init:
  - p: foo
  - or
  - p: bar
properties:
  a: !number { p: foo }
  b:
   type: boolean
   rule: { p: foo }`, NewSchema(ObjectType).
				SetInit(NewOr(NewSteps("p", "foo"), NewSteps("p", "bar"))).
				AddProperty("a", *NewSchema(NumberType).
					SetRule(NewSteps("p", "foo"))).
				AddProperty("b", *NewSchema(BooleanType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
properties:
  a: !boolean { p: &p foo }
  b: { p: *p }`, NewSchema(ObjectType).
				AddProperty("a", *NewSchema(BooleanType).
					SetRule(NewSteps("p", "foo"))).
				AddProperty("b", *NewSchema(StringType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
properties:
  a: &a
   type: boolean
   rule: { p: foo }
  b: *a`, NewSchema(ObjectType).
				AddProperty("a", *NewSchema(BooleanType).
					SetRule(NewSteps("p", "foo"))).
				AddProperty("b", *NewSchema(BooleanType).
					SetRule(NewSteps("p", "foo"))),
		},
		{
			`
type: object
properties:
  a:
   type: array
   rule: &a
     - { p: foo }
     - and
     - { p: foo }
  b:
   type: array
   rule: *a`, NewSchema(ObjectType).
				AddProperty("a", *NewSchema(ArrayType).
					SetRule(NewAnd(NewSteps("p", "foo"), NewSteps("p", "foo")))).
				AddProperty("b", *NewSchema(ArrayType).
					SetRule(NewAnd(NewSteps("p", "foo"), NewSteps("p", "foo")))),
		},
	}

	for i, test := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			s := new(Schema)
			err := yaml.Unmarshal([]byte(test.Yaml), s)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.Schema, s)
		})
	}
}

type testParser struct{}

func (t *testParser) GetString(_ *plugin.Context, content any, arg string) (string, error) {
	if str, ok := content.(string); ok {
		if str == arg {
			return str, nil
		}
	}
	return "", nil
}

func (t *testParser) GetStrings(_ *plugin.Context, content any, arg string) ([]string, error) {
	if str, ok := content.(string); ok {
		if str == arg {
			return []string{str}, nil
		}
	}
	return nil, nil
}

func (t *testParser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return t.GetString(ctx, content, arg)
}

func (t *testParser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return t.GetStrings(ctx, content, arg)
}

type unknown struct{ act Action }

func (u *unknown) UnmarshalYAML(value *yaml.Node) (err error) {
	u.act, err = actionDecode(value)
	return
}

func TestActions(t *testing.T) {
	t.Parallel()

	parser.Register("act", new(testParser))
	ctx := plugin.NewContext(plugin.Options{Timeout: time.Second})

	testCases := []struct {
		acts, content string
		want          any
		str           bool
	}{
		{
			`
- act: 1
- and
- act: 1
`, `1`, `11`, true,
		},
		{
			`
- act: 1
- and
- act: 1
- and
- act: 1
`, `1`, `111`, true,
		},
		{
			`
- act: 1
- and
- act: 1
`, `1`, []string{`1`, `1`}, false,
		},
		{
			`
- act: 2
- or
- act: 1
`, `1`, `1`, true,
		},
		{
			`
- act: 2
- or
- act: 2
- or
- act: 1
`, `1`, `1`, true,
		},
		{
			`
- act: 2
- or
- act: 1
`, `1`, []string{`1`}, false,
		},
		{
			`
- act: 1
- or
- act: 2
- and
- act: 1
`, `1`, `11`, true,
		},
		{
			`
- act: 1
- and
- act: 2
- or
- act: 1
`, `1`, `1`, true,
		},
		{
			`
- act: 1
- and
- act: 2
- or
- act: 1
`, `1`, []string{`1`}, false,
		},
		{
			`
- - act: 1
  - and
  - act: 1
- and
- act: 2
- or
- - act: 2
  - or
  - act: 1
`, `1`, `11`, true,
		},
		{
			`
- - act: 2
  - or
  - act: 1
- and
- act: 1
- or
- - act: 2
  - and
  - act: 1
`, `1`, `11`, true,
		},
		{
			`
- - act: 2
  - or
  - act: 1
- or
- - act: 1
  - and
  - act: 1
`, `1`, `1`, true,
		},
		{
			`
- - act: 2
  - and
  - act: 2
- or
- - act: 2
  - or
  - act: 1
`, `1`, `1`, true,
		},
		{
			`
- - act: 1
  - and
  - act: 1
- and
- act: 1
- and
- - act: 1
  - and
  - act: 1
`, `1`, `11111`, true,
		},
		{
			`
- - act: 1
  - and
  - act: 1
- and
- act: 1
- and
- - act: 1
  - and
  - act: 1
`, `1`, []string{`1`, `1`, `1`, `1`, `1`}, false,
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			u := new(unknown)
			err := yaml.Unmarshal([]byte(testCase.acts), u)
			if err != nil {
				t.Fatal(err)
			}
			var result any
			if testCase.str {
				result, err = GetString(u.act, ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			} else {
				result, err = GetStrings(u.act, ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			}
			assert.Equal(t, testCase.want, result, testCase.acts)
		})
	}
}
