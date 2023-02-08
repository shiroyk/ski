package schema

import (
	"fmt"
	"strings"

	"github.com/shiroyk/cloudcat/schema/parsers"
)

// Action The Schema Action
type Action struct {
	operator Operator
	step     []Step
}

// MarshalYAML encodes the action to yaml
func (a Action) MarshalYAML() (any, error) {
	if a.operator != OperatorNil {
		return a.operator, nil
	}
	if len(a.step) == 1 {
		return a.step[0].ToMap(), nil
	}
	var slice []any
	for _, step := range a.step {
		slice = append(slice, step.ToMap())
	}
	return slice, nil
}

// NewAction returns a new Action with the given Step
func NewAction(step ...Step) Action {
	return Action{step: step}
}

// NewActionOp returns a new Action with the given Operator
func NewActionOp(op Operator) Action {
	return Action{operator: op}
}

// toActionOp parsers the Operator string returns an operator Action
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

// ToMap returns a map of Step
func (s *Step) ToMap() map[string]string {
	return map[string]string{s.parser: s.rule}
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
func (a Actions) GetString(ctx *parsers.Context, content any) (string, error) {
	return runActions(a, ctx, content,
		func(p parsers.Parser) func(*parsers.Context, any, string) (string, error) {
			return p.GetString
		},
		func(s string) bool {
			return len(s) == 0
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetStrings run the action returns a slice of string
func (a Actions) GetStrings(ctx *parsers.Context, content any) ([]string, error) {
	return runActions(a, ctx, content,
		func(p parsers.Parser) func(*parsers.Context, any, string) ([]string, error) {
			return p.GetStrings
		},
		func(s []string) bool {
			return len(s) == 0
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// GetElement run the action returns an element string
func (a Actions) GetElement(ctx *parsers.Context, content any) (string, error) {
	return runActions(a, ctx, content,
		func(p parsers.Parser) func(*parsers.Context, any, string) (string, error) {
			return p.GetElement
		},
		func(s string) bool {
			return len(s) == 0
		},
		func(s1 string, s2 string) string {
			return s1 + s2
		})
}

// GetElements run the action returns a slice of element string
func (a Actions) GetElements(ctx *parsers.Context, content any) ([]string, error) {
	return runActions(a, ctx, content,
		func(p parsers.Parser) func(*parsers.Context, any, string) ([]string, error) {
			return p.GetElements
		},
		func(s []string) bool {
			return len(s) == 0
		},
		func(s1 []string, s2 []string) []string {
			return append(s1, s2...)
		})
}

// runActions runs the Actions
func runActions[T any](
	action Actions,
	ctx *parsers.Context,
	content any,
	runFn func(parsers.Parser) func(*parsers.Context, any, string) (T, error),
	orFn func(T) bool,
	andFn func(T, T) T,
) (ret T, err error) {
	var op Operator
	var result []T
	for _, act := range action {
		if act.operator != OperatorNil {
			op = act.operator
			continue
		}
		stepResult := content
		for _, step := range act.step {
			p, ok := parsers.GetParser(step.parser)
			if !ok {
				return ret, fmt.Errorf("schema %s not found", step.parser)
			}
			stepResult, err = runFn(p)(ctx, stepResult, step.rule)
			if err != nil {
				return
			}
		}
		if sr, ok := stepResult.(T); ok {
			if op == OperatorOr && orFn(sr) {
				break
			}
			result = append(result, sr)
		}
	}
	for _, s := range result {
		ret = andFn(ret, s)
	}
	return
}
