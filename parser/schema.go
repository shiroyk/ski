package parser

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
	OperatorNil Operator = ""
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

type Actions []Action

func (a Actions) GetString(ctx *Context, content any) (string, error) {
	return runActions(a, ctx, content, func(p Parser) func(*Context, any, string) (string, error) {
		return p.GetString
	})
}

func (a Actions) GetStrings(ctx *Context, content any) ([]string, error) {
	return runActions(a, ctx, content, func(p Parser) func(*Context, any, string) ([]string, error) {
		return p.GetStrings
	})
}
func (a Actions) GetElement(ctx *Context, content any) (string, error) {
	return runActions(a, ctx, content, func(p Parser) func(*Context, any, string) (string, error) {
		return p.GetElement
	})
}

func (a Actions) GetElements(ctx *Context, content any) ([]string, error) {
	return runActions(a, ctx, content, func(p Parser) func(*Context, any, string) ([]string, error) {
		return p.GetElements
	})
}

func runActions[T any](
	action []Action,
	ctx *Context,
	content any,
	runFn func(Parser) func(*Context, any, string) (T, error),
) (ret T, err error) {
	each := content
	for _, act := range action {
		for _, step := range act.Step {
			for _, r := range step.Rule {
				if p, ok := GetParser(step.Parser); ok {
					each, err = runFn(p)(ctx, each, r)
					if err != nil {
						return
					}
				} else {
					return ret, fmt.Errorf("parser %s not found", step.Parser)
				}
			}
		}
		if act.Operator == OperatorOr && reflect.ValueOf(each).IsZero() {
			break
		}
	}
	return each.(T), nil
}

type Schema struct {
	Type       Type
	Format     Type
	Init       Actions
	Rule       Actions
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
	return schema.AddOpInit(OperatorNil, step...)
}

func (schema *Schema) AddOpInit(op Operator, step ...Step) *Schema {
	if schema.Init == nil {
		schema.Init = make([]Action, 0)
	}

	schema.Init = append(schema.Init, NewOpAction(op, step...))

	return schema
}

func (schema *Schema) SetRule(action ...Action) *Schema {
	schema.Rule = action
	return schema
}

func (schema *Schema) AddRule(step ...Step) *Schema {
	return schema.AddOpRule(OperatorNil, step...)
}

func (schema *Schema) AddOpRule(op Operator, step ...Step) *Schema {
	if schema.Rule == nil {
		schema.Rule = make([]Action, 0)
	}

	schema.Rule = append(schema.Rule, NewOpAction(op, step...))

	return schema
}

func (schema *Schema) CloneWithType(typ Type) *Schema {
	return &Schema{
		Type:   typ,
		Format: schema.Format,
		Rule:   schema.Rule,
	}
}

type Source struct {
	Name    string
	BaseURL string
	Timeout time.Duration
	Header  map[string]string
}
