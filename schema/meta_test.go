package schema

import (
	"reflect"
	"strconv"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYaml(t *testing.T) {
	t.Parallel()
	r := new(Meta)

	testCases := []struct {
		Yaml   string
		Schema *Schema
	}{
		{`
title: { gq: foo }`, NewSchema(ObjectType).
			AddProperty("title", *NewSchema(StringType).
				AddRule(NewStep("gq", "foo"))),
		},
		{`
title:
  - gq: foo
  - gq: bar
  - gq: title`, NewSchema(ObjectType).
			AddProperty("title", *NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRule(NewStep("gq", "bar")).
				AddRule(NewStep("gq", "title"))),
		},
		{`
title:
  - gq: foo
  - or
  - gq: title`, NewSchema(ObjectType).
			AddProperty("title", *NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "title"))),
		},
		{`
body:
  - gq: foo
  - and
  - or
  - gq: body`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(StringType).
				AddRule(NewStep("gq", "foo")).
				AddRuleOp(OperatorAnd).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "body"))),
		},
		{`
body:
  - - gq: foo
    - gq: bar
  - or
  - - gq: title
    - gq: body`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(StringType).
				AddRule(NewStep("gq", "foo"), NewStep("gq", "bar")).
				AddRuleOp(OperatorOr).
				AddRule(NewStep("gq", "title"), NewStep("gq", "body"))),
		},
		{`
body:
  type: object
  properties:
    context:
      type: string
      format: boolean
      rule: { gq: foo }`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(ObjectType).
				AddProperty("context", *NewSchema(StringType, BooleanType).
					AddRule(NewStep("gq", "foo")))),
		},
		{`
body:
  type: array
  init: { gq: foo }
  properties:
    context:
      type: string
      format: integer
      rule: { gq: foo }`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(ArrayType).
				AddInit(NewStep("gq", "foo")).
				AddProperty("context", *NewSchema(StringType, IntegerType).
					AddRule(NewStep("gq", "foo")))),
		},
		{`
body:
  type: object
  init: { gq: foo }
  properties:
    context:
      type: number
      rule: { gq: foo }`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(ObjectType).
				AddInit(NewStep("gq", "foo")).
				AddProperty("context", *NewSchema(NumberType).
					AddRule(NewStep("gq", "foo")))),
		},
		{`
body:
  type: object
  init:
    - gq: foo
    - or
    - gq: bar
  properties:
    context:
      type: number
      rule: { gq: foo }`, NewSchema(ObjectType).
			AddProperty("body", *NewSchema(ObjectType).
				AddInit(NewStep("gq", "foo")).
				AddInitOp(OperatorOr).
				AddInit(NewStep("gq", "bar")).
				AddProperty("context", *NewSchema(NumberType).
					AddRule(NewStep("gq", "foo")))),
		},
	}

	for i, test := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := yaml.Unmarshal([]byte(test.Yaml), r)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r.Schema, test.Schema) {
				t.Error("not equal")
			}
		})
	}

}
