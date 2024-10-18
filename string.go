package ski

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cast"
)

// String the Executor for string value
type String string

func (k String) Exec(_ context.Context, _ any) (any, error) { return k.String(), nil }

func (k String) String() string { return string(k) }

type _str_join string

// str_join join strings
func str_join(args Arguments) (Executor, error) {
	return _str_join(args.GetString(0)), nil
}

func (sep _str_join) Exec(_ context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case []string:
		return strings.Join(s, string(sep)), nil
	case string:
		return s, nil
	case fmt.Stringer:
		return s.String(), nil
	default:
		v := reflect.ValueOf(arg)
		if v.Kind() != reflect.Slice {
			return nil, fmt.Errorf("expected string or []string, but got type %T", arg)
		}
		ret := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			s, err := cast.ToStringE(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			ret = append(ret, s)
		}
		return strings.Join(ret, string(sep)), nil
	}
}

type _str_split string

// str_split split string with separator
func str_split(args Arguments) (Executor, error) { return _str_split(args.GetString(0)), nil }

func (sep _str_split) Exec(_ context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case fmt.Stringer:
		return strings.Split(s.String(), string(sep)), nil
	case string:
		return strings.Split(s, string(sep)), nil
	case []string:
		return s, nil
	default:
		return nil, fmt.Errorf("expected string, but got type %T", arg)
	}
}

type _str_suffix string

// str_suffix string append suffix
func str_suffix(args Arguments) (Executor, error) { return _str_suffix(args.GetString(0)), nil }

func (suffix _str_suffix) Exec(_ context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case fmt.Stringer:
		return s.String() + string(suffix), nil
	case string:
		return s + string(suffix), nil
	case []string:
		ret := make([]string, 0, len(s))
		for _, v := range s {
			ret = append(ret, v+string(suffix))
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("expected string, but got type %T", arg)
	}
}

type _str_prefix string

// str_prefix string append prefix
func str_prefix(args Arguments) (Executor, error) { return _str_prefix(args.GetString(0)), nil }

func (prefix _str_prefix) Exec(_ context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case fmt.Stringer:
		return string(prefix) + s.String(), nil
	case string:
		return string(prefix) + s, nil
	case []string:
		ret := make([]string, 0, len(s))
		for _, v := range s {
			ret = append(ret, string(prefix)+v)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("expected string, but got type %T", arg)
	}
}
