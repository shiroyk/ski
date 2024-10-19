package ski

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// If control flow, check the condition is met.
// see Pipe, each, mapping, list_of, if_contains
type If interface {
	If(context.Context, any) (met bool)
}

// if_contains check string contains the substring,
// if met execute the Executor else return the original arg.
func if_contains(args Arguments) (Executor, error) {
	if len(args) == 0 {
		return new(_if_contains), nil
	}
	return &_if_contains{exec: Pipe(args)}, nil
}

type _if_contains struct {
	sub  string
	exec Executor
}

// Meta compile the regex expression from the tag
func (c *_if_contains) Meta(_, v *yaml.Node) (err error) {
	c.sub = strings.TrimPrefix(v.Tag, "!")
	return nil
}

func (c _if_contains) If(_ context.Context, arg any) bool {
	switch t := arg.(type) {
	case fmt.Stringer:
		return strings.Contains(t.String(), c.sub)
	case string:
		return strings.Contains(t, c.sub)
	case []string:
		for _, s := range t {
			ok := strings.Contains(s, c.sub)
			if ok {
				return true
			}
		}
	}
	return false
}

func (c _if_contains) Exec(ctx context.Context, arg any) (any, error) {
	if c.exec == nil {
		return arg, nil
	}
	return c.exec.Exec(ctx, arg)
}
