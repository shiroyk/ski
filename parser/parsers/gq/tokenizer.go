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

func parseRuleFunctions(rules string) (string, []ruleFunc, error) {
	funcs := make([]ruleFunc, 0)
	args := strings.Split(rules, "->")
	rule := strings.TrimSpace(args[0])

	if len(args) == 1 {
		return rule, funcs, nil
	}

	for _, fun := range args[1:] {
		fun = strings.TrimSpace(fun)
		if fun == "" {
			continue
		}
		f, err := parseFuncArguments(fun)
		if err != nil {
			return "", nil, err
		}
		if buildInFuncs[f.name] == nil {
			return "", nil, fmt.Errorf("unexpected not exists function %s", f.name)
		}
		funcs = append(funcs, *f)
	}

	return rule, funcs, nil
}

func parseFuncArguments(s string) (*ruleFunc, error) {
	openBracket := strings.IndexByte(s, '(')
	closeBracket := strings.LastIndexByte(s, ')')

	if openBracket == -1 {
		return &ruleFunc{name: s}, nil
	}

	if closeBracket == -1 {
		return nil, fmt.Errorf("unexpected function %s not close bracket", s)
	}

	funcName := s[0:openBracket]
	args := make([]string, 0)
	arg := strings.Builder{}
	state := commonState
	offset := openBracket + 1

	for offset < closeBracket {
		ch := s[offset]
		offset++
		switch ch {
		case '\\':
			offset++
			arg.WriteByte(s[offset])
			continue
		case '\'':
			if state == commonState {
				state = singleQuoteState
				continue
			} else if state == singleQuoteState {
				if offset > 1 && s[offset-2] == '\'' {
					args = append(args, "")
				}
				state = commonState
				continue
			}
		case '"':
			if state == commonState {
				state = doubleQuoteState
				continue
			} else if state == doubleQuoteState {
				if offset > 1 && s[offset-2] == '"' {
					args = append(args, "")
				}
				state = commonState
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
		return nil, fmt.Errorf("unexpected function %s argument quote not closed", s)
	}

	if arg.Len() > 0 {
		args = append(args, arg.String())
		arg.Reset()
	}

	return &ruleFunc{
		name: funcName,
		args: args,
	}, nil
}
