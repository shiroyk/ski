// Package regex the regexp parser
package regex

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
)

// Parser the regexp2 parser
type Parser struct{}

const key string = "regex"

func init() {
	parser.Register(key, new(Parser))
}

// GetString gets the string of the content with the given arguments.
// replace the string with the given regexp.
func (p Parser) GetString(_ *plugin.Context, content any, arg string) (string, error) {
	re, replace, start, count, err := parseRegexp(arg)
	if err != nil {
		return "", err
	}

	if str, ok := content.(string); ok {
		return re.Replace(str, replace, start, count)
	}

	return "", fmt.Errorf("unexpected content type %T", content)
}

// GetStrings gets the strings of the content with the given arguments.
// replace each string of the slice with the given regexp.
func (p Parser) GetStrings(_ *plugin.Context, content any, arg string) ([]string, error) {
	re, replace, start, count, err := parseRegexp(arg)
	if err != nil {
		return nil, err
	}

	if str, ok := content.([]string); ok {
		for i := 0; i < len(str); i++ {
			str[i], err = re.Replace(str[i], replace, start, count)
			if err != nil {
				return nil, err
			}
		}
		return str, nil
	}

	return nil, fmt.Errorf("unexpected content type %T", content)
}

// GetElement gets the element of the content with the given arguments.
// sames as GetString.
func (p Parser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// sames as GetStrings.
func (p Parser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
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

//nolint:gocognit
func parseRegexp(arg string) (re *regexp2.Regexp, replace string, start, count int, err error) {
	state := commonState
	pattern := strings.Builder{}
	start = -1
	count = -1
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
				return nil, "", start, count, fmt.Errorf("/ character must escaped")
			}
		}
	}

	if pattern.Len() > 0 {
		s1, s2, _ := strings.Cut(pattern.String(), ",")
		start, err = strconv.Atoi(s1)
		if err != nil {
			start = -1
		}
		count, err = strconv.Atoi(s2)
		if err != nil {
			count = -1
		}
	}

	return regexp2.MustCompile(regex, regexp2.RegexOptions(reOpt)), replace, start, count, nil
}
