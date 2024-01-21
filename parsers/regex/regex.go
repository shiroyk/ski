// Package regex the regexp parser
package regex

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/shiroyk/ski"
)

// Parser the regexp2 parser
type Parser struct{}

func init() {
	ski.Register("regex", new(Parser))
}

func (p Parser) Value(arg string) (ski.Executor, error) {
	ret, err := compile(arg)
	if err != nil {
		return nil, err
	}
	ret.exec = ret.string
	return ret, nil
}

func (p Parser) Element(arg string) (ski.Executor, error) { return p.Value(arg) }

func (p Parser) Elements(arg string) (ski.Executor, error) {
	ret, err := compile(arg)
	if err != nil {
		return nil, err
	}
	ret.exec = ret.strings
	return ret, nil
}

type regexp struct {
	re           *regexp2.Regexp
	start, count int
	replace      string
	exec         func(any) (any, error)
}

func (r regexp) Exec(_ context.Context, arg any) (any, error) { return r.exec(arg) }

func (r regexp) string(arg any) (any, error) {
	switch conv := arg.(type) {
	case string:
		return r.re.Replace(conv, r.replace, r.start, r.count)
	case []string:
		var err error
		for i := 0; i < len(conv); i++ {
			conv[i], err = r.re.Replace(conv[i], r.replace, r.start, r.count)
			if err != nil {
				return nil, err
			}
		}
		return conv, nil
	case fmt.Stringer:
		return r.re.Replace(conv.String(), r.replace, r.start, r.count)
	default:
		return nil, fmt.Errorf("unexpected type %T", arg)
	}
}

func (r regexp) strings(arg any) (any, error) {
	var (
		str []string
		err error
	)
	switch conv := arg.(type) {
	case string:
		str = []string{conv}
	case []string:
		str = conv
	case fmt.Stringer:
		str = []string{conv.String()}
	default:
		return nil, fmt.Errorf("unexpected type %T", arg)
	}

	for i := 0; i < len(str); i++ {
		str[i], err = r.re.Replace(str[i], r.replace, r.start, r.count)
		if err != nil {
			return nil, err
		}
	}
	return str, nil
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

func compile(arg string) (ret regexp, err error) {
	state := commonState
	pattern := strings.Builder{}
	ret.start = -1
	ret.count = -1
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
				ret.replace = pattern.String()
				pattern.Reset()
			default:
				return ret, fmt.Errorf("/ character must escaped")
			}
		}
	}

	if pattern.Len() > 0 {
		s1, s2, _ := strings.Cut(pattern.String(), ",")
		ret.start, err = strconv.Atoi(s1)
		if err != nil {
			ret.start = -1
			err = nil
		}
		ret.count, err = strconv.Atoi(s2)
		if err != nil {
			ret.count = -1
			err = nil
		}
	}

	ret.re, err = regexp2.Compile(regex, regexp2.RegexOptions(reOpt))
	return
}
