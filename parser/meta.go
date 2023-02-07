package parser

import (
	"time"

	"github.com/spf13/cast"
)

type Config struct {
	Separator string
	Timeout   time.Duration
}

type Meta struct {
	Source *Source `yaml:"source"`
	Config *Config `yaml:"config"`
	Schema *Schema `yaml:"scheme"`
}

func (meta *Meta) UnmarshalYAML(unmarshal func(any) error) error {
	var maps map[string]any

	if err := unmarshal(&maps); err != nil {
		return err
	}

	meta.buildMeta(maps)

	return nil
}

func (meta *Meta) buildMeta(maps map[string]any) {
	if source, ok := maps["source"]; ok {
		meta.Source = source.(*Source)
	}

	if config, ok := maps["config"]; ok {
		meta.Config = config.(*Config)
	}

	if object, ok := maps["schema"]; ok {
		if object, ok := object.(map[string]any); ok {
			maps = object
		}
	}

	schema := NewSchema(ObjectType)
	for field, s := range maps {
		schema.AddProperty(field, *buildSchema(s))
	}

	meta.Schema = schema
}

func buildStep(object any) []Step {
	switch object := object.(type) {
	case []any:
		actions := make([]Step, 0)

		for _, step := range object {
			actions = append(actions, buildStep(step)...)
		}

		return actions
	case map[string]any:
		steps := make([]Step, 0)

		for parser, step := range object {
			if array, ok := step.([]string); ok {
				steps = append(steps, NewStep(parser, array...))
			} else {
				steps = append(steps, NewStep(parser, cast.ToString(step)))
			}
		}

		return steps
	}
	return nil
}

func buildAction(object any) []Action {
	switch object := object.(type) {
	case []any:
		actions := make([]Action, len(object))

		for i, action := range object {
			act := buildAction(action)
			if len(act) > 0 {
				actions[i] = act[0]
			}
		}

		return actions
	case map[string]any:
		if and, ok := object[string(OperatorAnd)]; ok {
			return []Action{NewOpAction(OperatorAnd, buildStep(and)...)}
		}

		if or, ok := object[string(OperatorOr)]; ok {
			return []Action{NewOpAction(OperatorOr, buildStep(or)...)}
		}

		return []Action{NewAction(buildStep(object)...)}
	}

	return nil
}

func buildSchema(object any) *Schema {
	switch object := object.(type) {
	case []any:
		schema := NewSchema(StringType)
		schema.SetRule(buildAction(object)...)
		return schema
	case map[string]any:
		if tp, ok := object["type"]; ok {
			schema := NewSchema(ParseType(tp.(string)))

			if format, ok := object["format"]; ok {
				schema.Format = ParseType(format.(string))
			}

			if init, ok := object["init"]; ok {
				schema.SetInit(buildAction(init)...)
			}

			if properties, ok := object["properties"]; ok {
				if properties, ok := properties.(map[string]any); ok {
					for field, s := range properties {
						schema.AddProperty(field, *buildSchema(s))
					}
				}
			}

			if rule, ok := object["rule"]; ok {
				schema.SetRule(buildAction(rule)...)
			}

			return schema
		} else {
			schema := NewSchema(StringType)
			schema.SetRule(buildAction(object)...)

			return schema
		}
	}

	return nil
}
