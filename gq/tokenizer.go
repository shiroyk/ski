package gq

import (
	"fmt"
	"strings"

	"github.com/shiroyk/ski"
)

type tokenState int

const (
	commonState tokenState = iota
	singleQuoteState
	doubleQuoteState
)

func parseFuncArguments(s string) (name string, args []ski.Executor, err error) {
	openBracket := strings.IndexByte(s, '(')
	closeBracket := strings.LastIndexByte(s, ')')

	if openBracket == -1 {
		return s, nil, nil
	}

	if closeBracket == -1 {
		return name, nil, fmt.Errorf("unexpected function %s not close bracket", s)
	}

	name = s[0:openBracket]
	arg := strings.Builder{}
	state := commonState
	offset := openBracket + 1

	reverseState := func(s2 tokenState) bool {
		switch state {
		case commonState:
			state = s2
		case s2:
			if arg.Len() == 0 {
				arg.Grow(1)
			}
			state = commonState
		default:
			return false
		}
		return true
	}

	for offset < closeBracket {
		ch := s[offset]
		offset++
		switch ch {
		case '\\':
			offset++
			arg.WriteByte(s[offset])
			continue
		case '\'':
			if reverseState(singleQuoteState) {
				continue
			}
		case '"':
			if reverseState(doubleQuoteState) {
				continue
			}
		case ',':
			if state == commonState {
				args = append(args, ski.Raw(arg.String()))
				arg.Reset()
				continue
			}
		case ' ':
			if state == commonState {
				continue
			}
		}
		arg.WriteByte(ch)
	}

	if state == singleQuoteState || state == doubleQuoteState {
		return name, nil, fmt.Errorf("unexpected function %s argument quote not closed", s)
	}

	if arg.Cap() > 0 {
		args = append(args, ski.Raw(arg.String()))
		arg.Reset()
	}

	return
}

func isNumer(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
