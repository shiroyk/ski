package schema

import (
	"fmt"
	"time"

	"github.com/spf13/cast"
)

// Source the Meta source
type Source struct {
	Name    string
	BaseURL string
	Timeout time.Duration
	Header  map[string]string
}

// Meta the meta
type Meta struct {
	Source *Source `yaml:"source"`
	Schema *Schema `yaml:"scheme"`
}

// UnmarshalYAML decodes the Meta from yaml
func (meta *Meta) UnmarshalYAML(unmarshal func(any) error) error {
	var maps map[string]any

	if err := unmarshal(&maps); err != nil {
		return err
	}

	return meta.buildMeta(maps)
}

func (meta *Meta) buildMeta(maps map[string]any) (err error) {
	if source, ok := maps["source"]; ok {
		meta.Source = source.(*Source)
	}

	if properties, ok := maps["schema"].(map[string]any); ok {
		maps = properties
	}

	schema := NewSchema(ObjectType)
	for k, v := range maps {
		property, err := buildSchema(v)
		if err != nil {
			return err
		}
		schema.AddProperty(k, property)
	}

	meta.Schema = schema

	return
}

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
