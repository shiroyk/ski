package schema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shiroyk/cloudcat/parser"
	"gopkg.in/yaml.v3"
)

// Action The Schema Action
type Action struct {
	operator Operator
	step     []Step
}

// UnmarshalYAML decodes the Action from yaml
func (a *Actions) UnmarshalYAML(value *yaml.Node) (err error) {
	switch value.Kind {
	case yaml.MappingNode:
		steps, err := buildMapSteps(value)
		if err != nil {
			return err
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
			}

			if err != nil {
				return err
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
			return nil, errors.New("invalid step")
		}
	}
	return
}

// buildAction builds an Action for the slice Step
func buildAction(node *yaml.Node) (act Action, err error) {
	steps := make([]Step, 0, len(node.Content))
	for _, stepsNode := range node.Content {
		s, err := buildMapSteps(stepsNode)
		if err != nil {
			return act, err
		}
		steps = append(steps, s...)
	}
	return NewAction(steps...), nil
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
func (a *Actions) GetString(ctx *parser.Context, content any) (string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*parser.Context, any, string) (string, error) {
			return p.GetString
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetStrings run the action returns a slice of string
func (a *Actions) GetStrings(ctx *parser.Context, content any) ([]string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*parser.Context, any, string) ([]string, error) {
			return p.GetStrings
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// GetElement run the action returns an element string
func (a *Actions) GetElement(ctx *parser.Context, content any) (string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*parser.Context, any, string) (string, error) {
			return p.GetElement
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetElements run the action returns a slice of element string
func (a *Actions) GetElements(ctx *parser.Context, content any) ([]string, error) {
	return runActions(*a, ctx, content,
		func(p parser.Parser) func(*parser.Context, any, string) ([]string, error) {
			return p.GetElements
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// runActions runs the Actions
func runActions[T string | []string](
	action Actions,
	ctx *parser.Context,
	content any,
	runFn func(parser.Parser) func(*parser.Context, any, string) (T, error),
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
