package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// ErrInvalidSchema invalid schema error
var ErrInvalidSchema = errors.New("invalid schema")

// ErrAliasRecursive invalid alias error
var ErrAliasRecursive = errors.New("alias can't be recursive")

// ErrInvalidStep invalid step error
var ErrInvalidStep = errors.New("invalid step")

// Type The property type.
type Type string

const (
	// StringType The Type of string.
	StringType Type = "string"
	// NumberType The Type of number.
	NumberType Type = "number"
	// IntegerType The Type of integer.
	IntegerType Type = "integer"
	// BooleanType The Type of boolean.
	BooleanType Type = "boolean"
	// ObjectType The Type of object.
	ObjectType Type = "object"
	// ArrayType The Type of array.
	ArrayType Type = "array"
)

// ToType parses the schema type.
func ToType(s any) (Type, error) {
	switch s {
	case "string":
		return StringType, nil
	case "array":
		return ArrayType, nil
	case "object":
		return ObjectType, nil
	case "number":
		return NumberType, nil
	case "integer":
		return IntegerType, nil
	case "boolean":
		return BooleanType, nil
	}
	return "", fmt.Errorf("invalid type %s", s)
}

// Operator The Action operator.
type Operator string

const (
	// OperatorNil The Operator of nil.
	OperatorNil Operator = ""
	// OperatorAnd The Operator of and.
	// Action result A, B; Join the A + B.
	OperatorAnd Operator = "and"
	// OperatorOr The Operator of or.
	// Action result A, B; if result A is nil return B.
	OperatorOr Operator = "or"
)

// Schema The schema.
type Schema struct {
	Type       Type     `yaml:"type"`
	Format     Type     `yaml:"format,omitempty"`
	Init       Actions  `yaml:"init,omitempty"`
	Rule       Actions  `yaml:"rule,omitempty"`
	Properties Property `yaml:"properties,omitempty"`
}

// Property The Schema property.
type Property map[string]Schema

// NewSchema returns a new Schema with the given Type.
// The first argument is the Schema.Type, second is the Schema.Format.
func NewSchema(types ...Type) *Schema {
	switch {
	case len(types) == 0:
		panic("schema must have type")
	case len(types) == 1:
		return &Schema{
			Type: types[0],
		}
	default:
		return &Schema{
			Type:   types[0],
			Format: types[1],
		}
	}
}

// SetProperty set the Property to Schema.Properties.
func (schema *Schema) SetProperty(m Property) *Schema {
	schema.Properties = m
	return schema
}

// AddProperty append a field string with Schema to Schema.Properties.
func (schema *Schema) AddProperty(field string, s Schema) *Schema {
	if schema.Properties == nil {
		property := make(map[string]Schema)
		schema.Properties = property
	}

	schema.Properties[field] = s

	return schema
}

// SetInit set the Init Action to Schema.Init.
func (schema *Schema) SetInit(action []Action) *Schema {
	schema.Init = action
	return schema
}

// AddInit append Step to Schema.Init.
func (schema *Schema) AddInit(step ...Step) *Schema {
	schema.Init = append(schema.Init, NewAction(step...))
	return schema
}

// AddInitOp append Operator to Schema.Init.
func (schema *Schema) AddInitOp(op Operator) *Schema {
	schema.Init = append(schema.Init, NewActionOp(op))
	return schema
}

// SetRule set the Init Action to Schema.Rule.
func (schema *Schema) SetRule(action []Action) *Schema {
	schema.Rule = action
	return schema
}

// AddRule append Step to Schema.Init.
func (schema *Schema) AddRule(step ...Step) *Schema {
	schema.Rule = append(schema.Rule, NewAction(step...))
	return schema
}

// AddRuleOp append Operator to Schema.Rule.
func (schema *Schema) AddRuleOp(op Operator) *Schema {
	schema.Rule = append(schema.Rule, NewActionOp(op))
	return schema
}

// CloneWithType returns a copy of Schema.
// Schema.Format and Schema.Rule will be copied.
func (schema *Schema) CloneWithType(typ Type) *Schema {
	return &Schema{
		Type:   typ,
		Format: schema.Format,
		Rule:   schema.Rule,
	}
}

// UnmarshalYAML decodes the Schema from yaml
func (schema *Schema) UnmarshalYAML(node *yaml.Node) (err error) {
	*schema, err = buildSchema(node)
	return
}

// buildSchema builds a Schema
func buildSchema(node *yaml.Node) (schema Schema, err error) {
	switch node.Kind {
	case yaml.SequenceNode:
		return buildStringSchema(node)
	case yaml.MappingNode:
		typed := slices.ContainsFunc(node.Content,
			func(node *yaml.Node) bool {
				return node.Value == "type"
			})
		if typed {
			return buildTypedSchema(node)
		}
		return buildStringSchema(node)
	case yaml.AliasNode:
		if node.Value == node.Alias.Anchor {
			return schema, ErrAliasRecursive
		}
		return buildSchema(node.Alias)
	default:
		err = ErrInvalidSchema
	}
	return
}

// buildStringSchema builds a StringType Schema
func buildStringSchema(node *yaml.Node) (schema Schema, err error) {
	var acts Actions
	if err = node.Decode(&acts); err == nil {
		schema.Type = StringType
		schema.Rule = acts
	}
	return
}

// buildTypedSchema builds a specific Type Schema
//
//nolint:nakedret
func buildTypedSchema(node *yaml.Node) (schema Schema, err error) {
	for i := 0; i < len(node.Content); i += 2 {
		field, value := node.Content[i], node.Content[i+1]
		switch field.Value {
		case "type":
			schema.Type, err = ToType(value.Value)
		case "format":
			schema.Format, err = ToType(value.Value)
		case "init":
			var acts Actions
			err = value.Decode(&acts)
			schema.Init = acts
		case "rule":
			var acts Actions
			err = value.Decode(&acts)
			schema.Rule = acts
		case "properties":
			schema.Properties = make(Property, len(value.Content)/2)
			for j := 0; j < len(value.Content); j += 2 {
				k, v := value.Content[j], value.Content[j+1]
				schema.Properties[k.Value], err = buildSchema(v)
				if err != nil {
					return
				}
			}
		}
		if err != nil {
			return
		}
	}

	return
}

// MarshalYAML encodes the Schema
func (schema Schema) MarshalYAML() (any, error) {
	if schema.Type == StringType &&
		len(schema.Init) == 0 &&
		len(schema.Rule) > 0 {
		return schema.Rule, nil
	}
	s := make(map[string]any, 5)
	s["type"] = schema.Type
	if schema.Format != "" {
		s["format"] = schema.Format
	}
	if len(schema.Init) > 0 {
		s["init"] = schema.Init
	}
	if len(schema.Rule) > 0 {
		s["rule"] = schema.Rule
	}
	if len(schema.Properties) > 0 {
		s["properties"] = schema.Properties
	}
	return s, nil
}

// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
func (schema Schema) MarshalText() ([]byte, error) {
	if schema.Type == "" {
		return nil, nil
	}
	return yaml.Marshal(schema)
}

// UnmarshalText must be able to decode the form generated by MarshalText.
func (schema *Schema) UnmarshalText(text []byte) error {
	return yaml.Unmarshal(text, schema)
}

// Action The Schema Action
type Action struct {
	operator Operator
	step     []Step
}

// UnmarshalYAML decodes the Action from yaml
//
//nolint:nakedret
func (a *Actions) UnmarshalYAML(value *yaml.Node) (err error) {
	switch value.Kind { //nolint:exhaustive
	case yaml.MappingNode:
		var steps []Step
		steps, err = buildMapSteps(value)
		if err != nil {
			return
		}
		*a = []Action{NewAction(steps...)}
	case yaml.SequenceNode:
		*a = make(Actions, 0, len(value.Content))
		var act Action
		for _, node := range value.Content {
			switch node.Kind {
			case yaml.MappingNode:
				var steps []Step
				steps, err = buildMapSteps(node)
				act = NewAction(steps...)
			case yaml.ScalarNode:
				act, err = toActionOp(node.Value)
			case yaml.SequenceNode:
				act, err = buildAction(node)
			default:
				continue
			}

			if err != nil {
				return
			}

			*a = append(*a, act)
		}
	}
	return
}

// buildMapSteps builds a slice of Step from map
func buildMapSteps(node *yaml.Node) (steps []Step, err error) {
	steps = make([]Step, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if k.Kind == yaml.ScalarNode && v.Kind == yaml.ScalarNode {
			steps = append(steps, NewStep(k.Value, v.Value))
		} else {
			return nil, ErrInvalidStep
		}
	}
	return
}

// buildAction builds an Action for the slice Step
func buildAction(node *yaml.Node) (act Action, err error) {
	actSteps := make([]Step, 0, len(node.Content))
	for _, stepsNode := range node.Content {
		var steps []Step
		steps, err = buildMapSteps(stepsNode)
		if err != nil {
			return act, err
		}
		actSteps = append(actSteps, steps...)
	}
	return NewAction(actSteps...), nil
}

// MarshalYAML encodes the action to yaml
func (a Action) MarshalYAML() (any, error) {
	if a.operator != OperatorNil {
		return a.operator, nil
	}
	if len(a.step) == 1 {
		return a.step[0], nil
	}
	return a.step, nil
}

// NewAction returns a new Action with the given Step
func NewAction(step ...Step) Action {
	return Action{step: step}
}

// NewActionOp returns a new Action with the given Operator
func NewActionOp(op Operator) Action {
	return Action{operator: op}
}

// toActionOp parser the Operator string returns an operator Action
func toActionOp(op string) (ret Action, err error) {
	if op == string(OperatorAnd) || op == strings.ToUpper(string(OperatorAnd)) {
		return NewActionOp(OperatorAnd), nil
	}
	if op == string(OperatorOr) || op == strings.ToUpper(string(OperatorOr)) {
		return NewActionOp(OperatorOr), nil
	}

	return ret, fmt.Errorf("invalid operation %v", op)
}

// Step The Action of step
type Step struct {
	parser string
	rule   string
}

// MarshalYAML encodes to yaml
func (s Step) MarshalYAML() (any, error) {
	return map[string]string{s.parser: s.rule}, nil
}

// NewStep returns a new Step with the given params
func NewStep(parser string, rule string) Step {
	return Step{
		parser: parser,
		rule:   rule,
	}
}

// Actions slice of Action
type Actions []Action

// GetString run the action returns a string
func (a *Actions) GetString(ctx *plugin.Context, content any) (string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) (string, error) {
			return p.GetString
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetStrings run the action returns a slice of string
func (a *Actions) GetStrings(ctx *plugin.Context, content any) ([]string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) ([]string, error) {
			return p.GetStrings
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// GetElement run the action returns an element string
func (a *Actions) GetElement(ctx *plugin.Context, content any) (string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) (string, error) {
			return p.GetElement
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetElements run the action returns a slice of element string
func (a *Actions) GetElements(ctx *plugin.Context, content any) ([]string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) ([]string, error) {
			return p.GetElements
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// runActions runs the Actions
//
//nolint:nakedret
func runActions[T string | []string](
	action Actions,
	ctx *plugin.Context,
	content any,
	runFn func(parser.Parser) func(*plugin.Context, any, string) (T, error),
	andFn func(T, T) T,
) (ret T, err error) {
	var op Operator
	for _, act := range action {
		if act.operator != OperatorNil {
			op = act.operator
			continue
		}
		stepResult := content
		for _, step := range act.step {
			p, ok := parser.GetParser(step.parser)
			if !ok {
				return ret, fmt.Errorf("parser %s not found", step.parser)
			}
			stepResult, err = runFn(p)(ctx, stepResult, step.rule)
			if err != nil {
				return
			}
		}
		if sr, ok := stepResult.(T); ok {
			if op == OperatorOr && len(sr) == 0 {
				break
			}
			ret = andFn(ret, sr)
		}
	}

	return
}
