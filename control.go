package ski

import (
	"context"
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
	"gopkg.in/yaml.v3"
)

// If control flow, check the condition is met.
// see Pipe, each, mapping, list_of, if_match
type If interface {
	If(context.Context, any) (met bool)
}

// if_match check string match the regex expression,
// if met execute the Executor else return the original arg.
func if_match(args Arguments) (Executor, error) {
	if len(args) == 0 {
		return new(_if_match), nil
	}
	return &_if_match{exec: Pipe(args)}, nil
}

type _if_match struct {
	re   *regexp2.Regexp
	exec Executor
}

// Meta compile the regex expression from the tag
func (c *_if_match) Meta(_, v *yaml.Node) (err error) {
	c.re, err = regexp2.Compile(strings.TrimPrefix(v.Tag, "!"), regexp2.None)
	return err
}

func (c _if_match) If(_ context.Context, arg any) bool {
	switch t := arg.(type) {
	case fmt.Stringer:
		ok, _ := c.re.MatchString(t.String())
		return ok
	case string:
		ok, _ := c.re.MatchString(t)
		return ok
	case []string:
		for _, s := range t {
			ok, _ := c.re.MatchString(s)
			if ok {
				return true
			}
		}
	}
	return false
}

func (c _if_match) Exec(ctx context.Context, arg any) (any, error) {
	if c.exec == nil {
		return arg, nil
	}
	return c.exec.Exec(ctx, arg)
}
