// Package jq the json path parser
package jq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	"github.com/shiroyk/ski"
)

// Parser the json path parser
type Parser struct{}

func init() {
	ski.Register("jq", new(Parser))
}

func (p Parser) Value(arg string) (ski.Executor, error) {
	x, err := jp.ParseString(arg)
	if err != nil {
		return nil, err
	}
	return expr{x, x.Normal()}, nil
}

type expr struct {
	jp.Expr
	normal bool
}

func (e expr) Exec(_ context.Context, arg any) (any, error) {
	obj, err := doc(arg)
	if err != nil {
		return nil, err
	}
	if e.normal {
		return e.First(obj), nil
	}
	return e.Get(obj), nil
}

func doc(content any) (any, error) {
	switch data := content.(type) {
	default:
		return content, nil
	case fmt.Stringer:
		return oj.ParseString(data.String())
	case json.RawMessage:
		return oj.Parse(data)
	case []byte:
		return oj.Parse(data)
	case []string:
		if len(data) == 0 {
			return nil, nil
		}
		return oj.ParseString(data[0])
	case string:
		return oj.ParseString(data)
	}
}
