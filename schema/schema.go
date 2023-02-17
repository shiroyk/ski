package schema

import (
	"fmt"

	"github.com/spf13/cast"
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
func (schema *Schema) UnmarshalYAML(unmarshal func(any) error) error {
	var maps map[string]any

	if err := unmarshal(&maps); err != nil {
		return err
	}

	*schema = *NewSchema(ObjectType)

	for k, v := range maps {
		property, err := buildSchema(v)
		if err != nil {
			return err
		}
		schema.AddProperty(k, property)
	}

	return nil
}

// buildSchema builds a Schema
func buildSchema(object any) (schema Schema, err error) {
	switch obj := object.(type) {
	case []any:
		return buildStringSchema(obj)
	case map[string]any:
		if tp, ok := obj["type"]; ok {
			return buildTypedSchema(tp, obj)
		}
		return buildStringSchema(obj)
	default:
		return schema, fmt.Errorf("invalid schema %v", obj)
	}
}

// buildStringSchema builds a StringType Schema
func buildStringSchema(obj any) (schema Schema, err error) {
	schema = *NewSchema(StringType)
	var act []Action
	act, err = buildAction(obj)
	if err != nil {
		return schema, err
	}
	schema.SetRule(act)
	return
}

// buildTypedSchema builds a Type with Schema
func buildTypedSchema(typed any, obj map[string]any) (schema Schema, err error) {
	var schemaType Type
	schemaType, err = ToType(typed)
	if err != nil {
		return
	}
	schema = *NewSchema(schemaType)

	if format, ok := obj["format"]; ok {
		schema.Format, err = ToType(format)
		if err != nil {
			return
		}
	}

	if init, ok := obj["init"]; ok {
		var act []Action
		act, err = buildAction(init)
		if err != nil {
			return schema, err
		}
		schema.SetInit(act)
	}

	if rule, ok := obj["rule"]; ok {
		var act []Action
		act, err = buildAction(rule)
		if err != nil {
			return schema, err
		}
		schema.SetRule(act)
		return
	}

	if properties, ok := obj["properties"].(map[string]any); ok {
		for field, s := range properties {
			var property Schema
			property, err = buildSchema(s)
			if err != nil {
				return
			}
			schema.AddProperty(field, property)
		}
	}

	return
}

// buildSchema builds a slice of Action
func buildAction(object any) (acts []Action, err error) {
	switch obj := object.(type) {
	case []any:
		for _, action := range obj {
			if op, ok := action.(string); ok {
				opAct, err := toActionOp(op)
				if err != nil {
					return nil, err
				}
				acts = append(acts, opAct)
				continue
			}

			steps, err := buildStep(action)
			if err != nil {
				return nil, err
			}

			acts = append(acts, NewAction(steps...))
		}
		return acts, nil
	case map[string]any:
		steps, err := buildStep(obj)
		if err != nil {
			return nil, err
		}

		return []Action{NewAction(steps...)}, nil
	default:

		return nil, fmt.Errorf("invalid action %v", obj)
	}
}

// toSteps converts a map to slice of Step
func toSteps(object any) ([]Step, error) {
	switch obj := object.(type) {
	case map[string]any:
		steps := make([]Step, 0, len(obj))
		for parser, step := range obj {
			stepStr, err := cast.ToStringE(step)
			if err != nil {
				return nil, err
			}
			steps = append(steps, NewStep(parser, stepStr))
		}
		return steps, nil
	default:
		return nil, fmt.Errorf("invalid step %v", obj)
	}
}

// buildStep builds a slice of Step
func buildStep(object any) ([]Step, error) {
	switch obj := object.(type) {
	case []any:
		steps := make([]Step, 0, len(obj))

		for _, step := range obj {
			s, err := toSteps(step)
			if err != nil {
				return nil, err
			}
			steps = append(steps, s...)
		}

		return steps, nil
	default:
		return toSteps(obj)
	}
}
