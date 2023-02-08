package schema

import (
	"fmt"
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

// NewSchema returns a new Schema with the given SchemaType.
// The first argument is the type, second is the format.
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
