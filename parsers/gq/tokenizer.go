package gq

import (
	"fmt"
	"strings"
)

type tokenState int

const (
	commonState tokenState = iota
	singleQuoteState
	doubleQuoteState
)

type ruleFunc struct {
	name string
	args []string
}

func parseRuleFunctions(ruleStr string) (rule string, funcs []ruleFunc, err error) {
	ruleFuncs := strings.Split(ruleStr, "->")
	if len(ruleFuncs) == 1 {
		return ruleFuncs[0], funcs, nil
	}
	rule = strings.TrimSpace(ruleFuncs[0])

	for _, function := range ruleFuncs[1:] {
		function = strings.TrimSpace(function)
		if function == "" {
			continue
		}
		fn, err := parseFuncArguments(function)
		if err != nil {
			return "", nil, err
		}
		if funcMap[fn.name] == nil {
			return "", nil, fmt.Errorf("function %s not exists", fn.name)
		}
		funcs = append(funcs, fn)
	}

	return
}

func parseFuncArguments(s string) (ret ruleFunc, err error) {
	openBracket := strings.IndexByte(s, '(')
	closeBracket := strings.LastIndexByte(s, ')')

	if openBracket == -1 {
		return ruleFunc{name: s}, nil
	}

	if closeBracket == -1 {
		return ret, fmt.Errorf("unexpected function %s not close bracket", s)
	}

	funcName := s[0:openBracket]
	args := make([]string, 0)
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
				args = append(args, arg.String())
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
		return ret, fmt.Errorf("unexpected function %s argument quote not closed", s)
	}

	if arg.Cap() > 0 {
		args = append(args, arg.String())
		arg.Reset()
	}

	return ruleFunc{
		name: funcName,
		args: args,
	}, nil
}
