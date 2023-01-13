package meta

import (
	"fmt"
	"reflect"
	"time"
)

type Type string

const (
	StringType  Type = "string"
	ArrayType   Type = "array"
	ObjectType  Type = "object"
	NumberType  Type = "number"
	IntegerType Type = "integer"
	BooleanType Type = "boolean"
)

func ParseType(s string) Type {
	switch s {
	case "string":
		return StringType
	case "array":
		return ArrayType
	case "object":
		return ObjectType
	case "number":
		return NumberType
	case "integer":
		return IntegerType
	case "boolean":
		return BooleanType
	}
	panic(fmt.Errorf("unknown type %s", s))
}

type Operator string

const (
	OperatorAnd Operator = "and"
	OperatorOr  Operator = "or"
)

type Step struct {
	Parser string
	Rule   []string
}

func NewStep(parser string, rules ...string) Step {
	return Step{
		Parser: parser,
		Rule:   rules,
	}
}

type Action struct {
	Operator Operator
	Step     []Step
}

func NewAction(step ...Step) Action {
	return NewOpAction("", step...)
}

func NewOpAction(op Operator, step ...Step) Action {
	return Action{
		Operator: op,
		Step:     step,
	}
}

func ActEach[T any](
	action []Action,
	content any,
	covert func(any, string, string) (T, error),
) T {
	each := content
	var err error
	for _, act := range action {
		for _, step := range act.Step {
			for _, r := range step.Rule {
				if each, err = covert(each, step.Parser, r); err != nil {
					panic(fmt.Errorf("%s: %s", step.Parser, err))
				}
			}
		}
		if act.Operator == OperatorOr && reflect.ValueOf(each).IsZero() {
			break
		}
	}
	return each.(T)
}

type Schema struct {
	Type       Type
	Format     Type
	Init       []Action
	Rule       []Action
	Properties Property
}

type Property map[string]Schema

func NewSchema(types ...Type) *Schema {
	if len(types) == 0 {
		panic("schema must have type")
	} else if len(types) == 1 {
		return &Schema{
			Type: types[0],
		}
	} else {
		return &Schema{
			Type:   types[0],
			Format: types[1],
		}
	}
}

func (schema *Schema) SetPropertyByMap(m map[string]Schema) *Schema {
	schema.Properties = m
	return schema
}

func (schema *Schema) SetProperty(m Property) *Schema {
	schema.Properties = m
	return schema
}

func (schema *Schema) AddProperty(field string, s Schema) *Schema {
	if schema.Properties == nil {
		property := make(map[string]Schema)
		schema.Properties = property
	}

	schema.Properties[field] = s

	return schema
}

func (schema *Schema) SetInit(action ...Action) *Schema {
	schema.Init = action
	return schema
}

func (schema *Schema) AddInit(step ...Step) *Schema {
	return schema.AddOpInit(OperatorAnd, step...)
}

func (schema *Schema) AddOpInit(op Operator, step ...Step) *Schema {
	if schema.Init == nil {
		schema.Init = make([]Action, 0)
	}

	slice := append(schema.Init, NewOpAction(op, step...))
	schema.Init = slice

	return schema
}

func (schema *Schema) SetRule(action ...Action) *Schema {
	schema.Rule = action
	return schema
}

func (schema *Schema) AddRule(step ...Step) *Schema {
	return schema.AddOpRule(OperatorAnd, step...)
}

func (schema *Schema) AddOpRule(op Operator, step ...Step) *Schema {
	if schema.Rule == nil {
		schema.Rule = make([]Action, 0)
	}

	slice := append(schema.Rule, NewOpAction(op, step...))
	schema.Rule = slice

	return schema
}

type Source struct {
	Name    string
	BaseUrl string
	Timeout time.Duration
	Header  map[string]string
}
