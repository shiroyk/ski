// Package regex the regexp executor
package regex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/shiroyk/ski"
)

func init() {
	ski.Register("regex.replace", new_replace())
	ski.Register("regex.match", new_match())
	ski.Register("regex.assert", new_assert())
}

type tokenState int

const (
	commonState tokenState = iota
	searchState
	replaceState
	flagState
)

var reOptMap = map[string]regexp2.RegexOptions{
	"i": regexp2.IgnoreCase,
	"m": regexp2.Multiline,
	"n": regexp2.ExplicitCapture,
	"c": regexp2.Compiled,
	"s": regexp2.Singleline,
	"x": regexp2.IgnorePatternWhitespace,
	"r": regexp2.RightToLeft,
	"d": regexp2.Debug,
	"e": regexp2.ECMAScript,
	"u": regexp2.Unicode,
}

type _replace struct {
	*regexp2.Regexp
	replace      string
	start, count int
}

func new_replace() ski.NewExecutor {
	return ski.StringExecutor(func(str string) (ski.Executor, error) {
		re, replace, start, count, err := Compile(str)
		if err != nil {
			return nil, err
		}
		s, err := strconv.Atoi(start)
		if err != nil {
			s = -1
		}
		c, err := strconv.Atoi(count)
		if err != nil {
			c = -1
		}
		return _replace{re, replace, s, c}, nil
	})
}

func (r _replace) Exec(_ context.Context, arg any) (any, error) {
	switch conv := arg.(type) {
	case string:
		return r.Replace(conv, r.replace, r.start, r.count)
	case ski.Iterator:
		if conv.Len() == 0 {
			return nil, nil
		}
		_, ok := conv.At(0).(string)
		if !ok {
			return nil, fmt.Errorf("regex.replace unsupported type %T", arg)
		}
		ret := make([]string, 0, conv.Len())
		for i := 0; i < conv.Len(); i++ {
			v, err := r.Replace(conv.At(i).(string), r.replace, r.start, r.count)
			if err != nil {
				return nil, err
			}
			ret = append(ret, v)
		}
		return ret, nil
	case fmt.Stringer:
		return r.Replace(conv.String(), r.replace, r.start, r.count)
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("regex.replace unsupported type %T", arg)
	}
}

type _match struct {
	*regexp2.Regexp
	start, end int
}

func new_match() ski.NewExecutor {
	return ski.StringExecutor(func(str string) (ski.Executor, error) {
		re, _, start, end, err := Compile(str)
		if err != nil {
			return nil, err
		}
		s, _ := strconv.Atoi(start)
		e, _ := strconv.Atoi(end)
		return _match{re, s, e}, nil
	})
}

func (r _match) Exec(_ context.Context, arg any) (any, error) {
	var str string
	switch t := arg.(type) {
	case nil:
		return nil, nil
	case string:
		str = t
	case []string:
		str = t[0]
	case fmt.Stringer:
		str = t.String()
	default:
		return nil, fmt.Errorf("regex.match unsupported type %T", arg)
	}

	all := r.findAllString(str)
	if len(all) == 0 {
		return nil, nil
	}

	// get the groups from start to end
	start, end := r.start, r.end
	if start < 0 {
		start += len(all)
	}
	if end == math.MaxInt {
		end = len(all)
	} else if end < 0 {
		end += len(all)
	}
	if start > len(all) || end > len(all) || start < 0 || end < 0 {
		return nil, nil
	}
	if start >= end {
		return all[start], nil
	}

	return all[start : end+1], nil
}

func (r _match) findAllString(s string) []string {
	var matches []string
	m, _ := r.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = r.FindNextMatch(m)
	}
	return matches
}

type _assert struct {
	*regexp2.Regexp
	err error
}

func new_assert() ski.NewExecutor {
	return ski.StringExecutor(func(str string) (ski.Executor, error) {
		re, msg, _, _, err := Compile(str)
		if err != nil {
			return nil, err
		}
		var ret _assert
		ret.Regexp = re
		if len(msg) > 0 {
			ret.err = errors.New(msg)
		}
		return ret, nil
	})
}

func (r _assert) assert(str string) error {
	ok, err := r.MatchString(str)
	if err != nil {
		if r.err != nil {
			return r.err
		}
		return fmt.Errorf(`assert failed %s`, err)
	}
	if !ok {
		if r.err != nil {
			return r.err
		}
		return errors.New(`assert failed`)
	}
	return nil
}

func (r _assert) Exec(_ context.Context, arg any) (any, error) {
	var err error
	switch conv := arg.(type) {
	case string:
		err = r.assert(conv)
	case []string:
		for _, str := range conv {
			err = r.assert(str)
			if err != nil {
				return nil, err
			}
		}
	case fmt.Stringer:
		err = r.assert(conv.String())
	default:
		return nil, fmt.Errorf("regex.assert unsupported type %T", arg)
	}
	if err != nil {
		return nil, err
	}
	return arg, nil
}

// Compile the pattern `/regex/replace/options{start,count}` or `/regex/options{start,count}`
func Compile(arg string) (re *regexp2.Regexp, replace string, start, count string, err error) {
	state := commonState
	pattern := strings.Builder{}
	pattern.Grow(len(arg))
	var offset int
	var regex string
	var reOpt int32

	for offset < len(arg) {
		ch := arg[offset]
		offset++
		switch ch {
		default:
			if state == flagState {
				if i, ok := reOptMap[string(ch)]; ok {
					reOpt |= int32(i)
				} else if ch >= '0' && ch <= '9' || ch == '-' || ch == ',' {
					pattern.WriteByte(ch)
				}
			} else {
				pattern.WriteByte(ch)
			}
		case '\\':
			if nextCh := arg[offset]; nextCh == '/' {
				pattern.WriteByte(nextCh)
				offset++
			} else {
				pattern.WriteByte(ch)
			}
		case '/':
			switch state {
			case commonState:
				state = searchState
			case searchState:
				state = replaceState
				regex = pattern.String()
				pattern.Reset()
			case replaceState:
				state = flagState
				replace = pattern.String()
				pattern.Reset()
			default:
				err = fmt.Errorf("/ character must escaped")
				return
			}
		}
	}

	if state == replaceState && pattern.Len() > 0 {
		flags := pattern.String()
		pattern.Reset()
		for i := 0; i < len(flags); i++ {
			ch := flags[i]
			if f, ok := reOptMap[string(ch)]; ok {
				reOpt |= int32(f)
			} else if ch >= '0' && ch <= '9' || ch == '-' || ch == ',' {
				pattern.WriteByte(ch)
			}
		}
	}

	if pattern.Len() > 0 {
		start, count, _ = strings.Cut(pattern.String(), ",")
		pattern.Reset()
	}

	re, err = regexp2.Compile(regex, regexp2.RegexOptions(reOpt))
	return
}
