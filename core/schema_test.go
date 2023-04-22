package core

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
{ gq: foo }`, NewSchema(StringType).
				AddRule(NewStep("gq", "foo")),
		},
		{
			`
- gq: foo
- gq: bar
- gq: title`, NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRule(NewStep("gq", "bar")).
				AddRule(NewStep("gq", "title")),
		},
		{
			`
- gq: foo
- or
- gq: title`, NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "title")),
		},
		{
			`
- gq: foo
- and
- or
- gq: body`, NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRuleOp(OperatorAnd).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "body")),
		},
		{
			`
- - gq: foo
  - gq: bar
- or
- - gq: title
  - gq: body`, NewSchema(StringType).
				AddRule(NewStep("gq", "foo"), NewStep("gq", "bar")).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "title"), NewStep("gq", "body")),
		},
		{
			`
type: integer
rule: { gq: foo }`, NewSchema(IntegerType).
				AddRule(NewStep("gq", "foo")),
		},
		{
			`
type: number
rule: { gq: foo }`, NewSchema(NumberType).
				AddRule(NewStep("gq", "foo")),
		},
		{
			`
type: boolean
rule: { gq: foo }`, NewSchema(BooleanType).
				AddRule(NewStep("gq", "foo")),
		},
		{
			`
type: object
properties:
  context:
    type: string
    format: boolean
    rule: { gq: foo }`, NewSchema(ObjectType).
				AddProperty("context", *NewSchema(StringType, BooleanType).
					AddRule(NewStep("gq", "foo"))),
		},
		{
			`
type: array
init: { gq: foo }
properties:
  context:
    type: string
    format: integer
    rule: { gq: foo }`, NewSchema(ArrayType).
				AddInit(NewStep("gq", "foo")).
				AddProperty("context", *NewSchema(StringType, IntegerType).
					AddRule(NewStep("gq", "foo"))),
		},
		{
			`
type: object
init: { gq: foo }
properties:
  context:
    type: number
    rule: { gq: foo }`, NewSchema(ObjectType).
				AddInit(NewStep("gq", "foo")).
				AddProperty("context", *NewSchema(NumberType).
					AddRule(NewStep("gq", "foo"))),
		},
		{
			`
type: object
init:
  - gq: foo
  - or
  - gq: bar
properties:
  context:
    type: number
    rule: { gq: foo }`, NewSchema(ObjectType).
				AddInit(NewStep("gq", "foo")).
				AddInitOp(OperatorOr).
				AddInit(NewStep("gq", "bar")).
				AddProperty("context", *NewSchema(NumberType).
					AddRule(NewStep("gq", "foo"))),
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

func TestActions(t *testing.T) {
	t.Parallel()

	if _, ok := parser.GetParser("act"); !ok {
		parser.Register("act", new(testParser))
	}
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
- act: 1
`, `1`, []string{`1`}, false,
		},
		{
			`
- act: 1
- and
- act: 2
- or
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
`, `1`, []string{`1`, `1`}, false,
		},
		{
			`
- - act: 1
    act: 1
- and
- - act: 2
- or
- - act: 1
`, `1`, `11`, true,
		},
		{
			`
- - act: 1
    act: 1
- and
- - act: 2
- or
- - act: 1
`, `1`, []string{`1`}, false,
		},
	}
	if _, ok := parser.GetParser("act"); !ok {
		parser.Register("act", new(testParser))
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var act Actions
			err := yaml.Unmarshal([]byte(testCase.acts), &act)
			if err != nil {
				t.Error(err)
			}
			var result any
			if testCase.str {
				result, err = act.GetString(ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			} else {
				result, err = act.GetStrings(ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			}
			assert.Equal(t, testCase.want, result)
		})
	}
}

func TestAliasRecursive(t *testing.T) {
	c := `
type: object
properties:
  comment: &c
    type: array
    properties:
      content: {}
      replies: *c`
	s := new(Schema)
	err := yaml.Unmarshal([]byte(c), s)
	assert.ErrorIs(t, err, ErrAliasRecursive)
}
