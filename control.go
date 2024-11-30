package ski

import (
	"context"
	"fmt"
	"strings"
)

// if_contains check string contains the substring,
// if not contains return the ErrYield else return the original arg.
func if_contains(args Arguments) (Executor, error) {
	return _if_contains(args.GetString(0)), nil
}

type _if_contains string

func (s _if_contains) Exec(ctx context.Context, arg any) (any, error) {
	var ok bool
	switch v := arg.(type) {
	case string:
		ok = strings.Contains(v, string(s))
	case fmt.Stringer:
		ok = strings.Contains(v.String(), string(s))
	case []string:
		for _, i := range v {
			if ok = strings.Contains(i, string(s)); ok {
				return arg, nil
			}
		}
	}
	if ok {
		return arg, nil
	}
	return nil, ErrYield
}
