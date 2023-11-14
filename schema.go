package cloudcat

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"gopkg.in/yaml.v3"
)

var (
	// ErrInvalidSchema invalid schema error
	ErrInvalidSchema = errors.New("invalid schema")
	// ErrAliasRecursive invalid alias error
	ErrAliasRecursive = errors.New("alias can't be recursive")
	// ErrInvalidAction invalid action error
	ErrInvalidAction = errors.New("invalid action")
	// ErrInvalidStep invalid step error
	ErrInvalidStep = errors.New("invalid step")
)

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
	// OperatorAnd The Operator of and.
	// Action result A, B; Join the A + B.
	OperatorAnd Operator = "and"
	// OperatorOr The Operator of or.
	// Action result A, B; if result A is nil return B else return A.
	OperatorOr Operator = "or"
	// OperatorNot The Operator of not.
	// Action result A, B; if result A is not nil return B else return nil.
	OperatorNot Operator = "not"
)

// Schema The schema.
type Schema struct {
	Type       Type     `yaml:"type"`
	Format     Type     `yaml:"format,omitempty"`
	Init       Action   `yaml:"init,omitempty"`
	Rule       Action   `yaml:"rule,omitempty"`
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
func (schema *Schema) SetInit(action Action) *Schema {
	schema.Init = action
	return schema
}

// SetRule set the Init Action to Schema.Rule.
func (schema *Schema) SetRule(action Action) *Schema {
	schema.Rule = action
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
		return buildSchema(node.Alias)
	default:
		err = ErrInvalidSchema
	}
	return
}

// buildStringSchema builds a StringType Schema
func buildStringSchema(node *yaml.Node) (schema Schema, err error) {
	schema.Type = StringType
	schema.Rule, err = actionDecode(node)
	if err != nil {
		return
	}
	if len(node.Tag) > 2 && node.Tag[0] == '!' && node.Tag[1] != '!' {
		tags := strings.Split(node.Tag[1:], "/")
		schema.Type, err = ToType(tags[0])
		if len(tags) > 1 {
			schema.Format, err = ToType(tags[1])
		}
		if err != nil {
			return schema, fmt.Errorf("invalid tag %s", node.Tag)
		}
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
			schema.Init, err = actionDecode(value)
		case "rule":
			schema.Rule, err = actionDecode(value)
		case "properties":
			if len(value.Content) == 2 {
				schema.Properties = make(Property, 2)
				k, v := value.Content[0], value.Content[1]
				if k.Kind == yaml.MappingNode {
					schema.Properties["$key"], err = buildSchema(k)
					schema.Properties["$value"], err = buildSchema(v)
					return
				}
				schema.Properties[k.Value], err = buildSchema(v)
				return
			}
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
		schema.Init == nil {
		return schema.Rule, nil
	}
	s := make(map[string]any, 5)
	s["type"] = schema.Type
	if schema.Format != "" {
		s["format"] = schema.Format
	}
	if schema.Init != nil {
		s["init"] = schema.Init
	}
	if schema.Rule != nil {
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
type Action interface {
	// Left returns the left Action
	Left() Action
	// Right returns the right Action
	Right() Action
}

// Step The Action of step
type Step struct{ K, V string }

// MarshalYAML encodes to yaml
func (s Step) MarshalYAML() (any, error) {
	return map[string]string{s.K: s.V}, nil
}

// Steps slice of Step
type Steps []Step

// NewSteps return new Steps
func NewSteps(str ...string) *Steps {
	if len(str)%2 != 0 {
		panic(ErrInvalidStep)
	}
	steps := make(Steps, 0, len(str)/2)
	for i := 0; i < len(str); i += 2 {
		steps = append(steps, Step{str[i], str[i+1]})
	}
	return &steps
}

// Left returns the left Action
func (s *Steps) Left() Action { return nil }

// Right returns the right Action
func (s *Steps) Right() Action { return nil }

func (s *Steps) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		*s = make(Steps, 0, len(value.Content)/2)
		for i := 0; i < len(value.Content); i += 2 {
			k, v := value.Content[i], value.Content[i+1]
			if v.Kind == yaml.AliasNode {
				v = v.Alias
			}
			if k.Kind != yaml.ScalarNode || v.Kind != yaml.ScalarNode {
				return ErrInvalidStep
			}
			*s = append(*s, Step{k.Value, v.Value})
		}
	case yaml.SequenceNode:
		*s = make(Steps, 0, len(value.Content))
		for _, node := range value.Content {
			if node.Kind != yaml.MappingNode {
				return ErrInvalidStep
			}
			steps := new(Steps)
			if err := node.Decode(steps); err != nil {
				return err
			}
			*s = append(*s, *steps...)
		}
	default:
		return ErrInvalidStep
	}
	return nil
}

func (s Steps) MarshalYAML() (any, error) {
	if len(s) == 1 {
		return s[0], nil
	}
	return s, nil
}

// And Action node of Operator and
type And struct{ l, r Action }

// NewAnd create new And action with left and right Action
func NewAnd(left, right Action) *And    { return &And{left, right} }
func (a And) Left() Action              { return a.l }
func (a And) Right() Action             { return a.r }
func (a And) String() string            { return string(OperatorAnd) }
func (a And) MarshalYAML() (any, error) { return [...]any{a.l, a.String(), a.r}, nil }

// Or Action node of Operator or
type Or struct{ l, r Action }

// NewOr create new Or action with left and right Action
func NewOr(left, right Action) *Or     { return &Or{left, right} }
func (a Or) Left() Action              { return a.l }
func (a Or) Right() Action             { return a.r }
func (a Or) String() string            { return string(OperatorOr) }
func (a Or) MarshalYAML() (any, error) { return [...]any{a.l, a.String(), a.r}, nil }

// Not Action node of Operator not
type Not struct{ l, r Action }

// NewNot create new Not action with left and right Action
func NewNot(left, right Action) *Not    { return &Not{left, right} }
func (a Not) Left() Action              { return a.l }
func (a Not) Right() Action             { return a.r }
func (a Not) String() string            { return string(OperatorNot) }
func (a Not) MarshalYAML() (any, error) { return [...]any{a.l, a.String(), a.r}, nil }

// actionDecode decodes the Action from yaml.node
func actionDecode(value *yaml.Node) (ret Action, err error) {
	if value.Kind == yaml.DocumentNode {
		return nil, ErrInvalidAction
	}
	if value.Kind == yaml.AliasNode {
		value = value.Alias
	}
	multiStep := value.Kind == yaml.SequenceNode &&
		!slices.ContainsFunc(value.Content, func(e *yaml.Node) bool {
			return e.Kind == yaml.ScalarNode
		})
	if value.Kind == yaml.MappingNode || multiStep {
		steps := new(Steps)
		if err = value.Decode(steps); err != nil {
			return
		}
		return steps, nil
	}

	var op string
	var left Action
	for _, node := range value.Content {
		switch node.Kind {
		case yaml.MappingNode, yaml.SequenceNode:
			var leaf Action
			leaf, err = actionDecode(node)
			if err != nil {
				return
			}
			if left == nil {
				left = leaf
				continue
			}
			ret, err = toActionOp(op, left, leaf)
			left = nil
		case yaml.ScalarNode:
			op = node.Value
		default:
			continue
		}

		if err != nil {
			return
		}
	}

	if left != nil {
		return toActionOp(op, ret, left)
	}

	return
}

// toActionOp parser the Operator string returns an operator Action
func toActionOp(op string, left, right Action) (Action, error) {
	switch Operator(strings.ToLower(op)) {
	case OperatorAnd:
		return NewAnd(left, right), nil
	case OperatorOr:
		return NewOr(left, right), nil
	case OperatorNot:
		return NewNot(left, right), nil
	default:
		return nil, fmt.Errorf("invalid operator %v", op)
	}
}

// GetString run the action returns a string
func GetString(act Action, ctx *plugin.Context, content any) (string, error) {
	return runAction(act, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) (string, error) {
			return p.GetString
		},
		func(s1, s2 string) string {
			return s1 + s2
		})
}

// GetStrings run the action returns a slice of string
func GetStrings(act Action, ctx *plugin.Context, content any) ([]string, error) {
	return runAction(act, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) ([]string, error) {
			return p.GetStrings
		},
		func(s1, s2 []string) []string {
			return append(s1, s2...)
		})
}

// GetElement run the action returns an element string
func GetElement(act Action, ctx *plugin.Context, content any) (string, error) {
	return runAction(act, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) (string, error) {
			return p.GetElement
		},
		func(s1, s2 string) string {
			return s1 + s2
		})
}

// GetElements run the action returns a slice of element string
func GetElements(act Action, ctx *plugin.Context, content any) ([]string, error) {
	return runAction(act, ctx, content,
		func(p parser.Parser) func(*plugin.Context, any, string) ([]string, error) {
			return p.GetElements
		},
		func(s1, s2 []string) []string {
			return append(s1, s2...)
		})
}

// runAction runs the Action
//
//nolint:nakedret
func runAction[T string | []string](
	node Action,
	ctx *plugin.Context,
	content any,
	runFn func(parser.Parser) func(*plugin.Context, any, string) (T, error),
	joinFn func(T, T) T,
) (ret T, err error) {
	var stack []Action
	var left, _empty T
	var join bool

	for len(stack) > 0 || node != nil {
		// traverse the left subtree and push the node to the stack
		for node != nil {
			stack = append(stack, node)
			node = node.Left()
		}

		// pop the stack and process the node
		node = stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch node.(type) {
		case *Or:
			if len(left) > 0 {
				// discard right subtree if left subtree result is not empty
				node = nil
				if len(stack) == 0 {
					ret = joinFn(ret, left)
					left = _empty
				}
				continue
			}
			join = true
		case *And:
			if len(stack) == 0 {
				// join the left subtree result to ret
				ret = joinFn(ret, left)
				left = _empty
			}
			join = true
		case *Not:
			if len(left) == 0 {
				// discard right subtree if left subtree result is empty
				node = nil
				continue
			}
			left = _empty // discard left subtree result
			join = true
		case *Steps:
			result := content
			for _, step := range *node.(*Steps) {
				p, ok := parser.GetParser(step.K)
				if !ok {
					return ret, fmt.Errorf("parser %s not found", step.K)
				}
				result, err = runFn(p)(ctx, result, step.V)
				if err != nil {
					return
				}
			}

			switch {
			case len(stack) == 0:
				ret = joinFn(ret, result.(T))
				return
			case join:
				left = joinFn(left, result.(T))
				join = false
			default:
				left = result.(T)
			}
		}

		node = node.Right()
	}
	return
}
