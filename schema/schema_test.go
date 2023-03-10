package schema

import (
	"strconv"
	"testing"
	"time"

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

func TestSourceYaml(t *testing.T) {
	t.Parallel()
	s := `source:
  name: test
  http: |
    http://localhost
    user-agent: cloudcat
  timeout: 60s
`
	model := new(Model)
	err := yaml.Unmarshal([]byte(s), model)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, Source{
		Name:    "test",
		HTTP:    "http://localhost\nuser-agent: cloudcat\n",
		Timeout: time.Minute,
	}, *model.Source)
}
