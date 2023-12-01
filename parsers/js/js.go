// Package js the js parser
package js

import (
	"encoding/json"

	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/spf13/cast"
)

// Parser the js parser
type Parser struct{}

const key string = "js"

func init() {
	parser.Register(key, new(Parser))
}

// GetString gets the string of the content with the given arguments.
// returns the string result.
func (p *Parser) GetString(ctx *plugin.Context, content any, arg string) (string, error) {
	v, err := p.run(ctx, content, arg)
	if err != nil {
		return "", err
	}
	return toString(v)
}

// GetStrings gets the strings of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetStrings(ctx *plugin.Context, content any, arg string) ([]string, error) {
	v, err := p.run(ctx, content, arg)
	if err != nil {
		return nil, err
	}
	return toStrings(v)
}

// GetElement gets the element of the content with the given arguments.
// returns the string result.
func (p *Parser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

func (p *Parser) run(ctx *plugin.Context, content any, script string) (any, error) {
	ctx.SetValue("content", content)
	result, err := js.RunString(ctx, script)
	if err != nil {
		return nil, err
	}
	return js.Unwrap(result)
}

func toString(value any) (ret string, err error) {
	switch value.(type) {
	case map[string]any, []any:
		bytes, err := json.Marshal(value)
		if err != nil {
			return ret, err
		}
		return string(bytes), nil
	case nil:
		return ret, nil
	default:
		return cast.ToStringE(value)
	}
}

func toStrings(value any) (ret []string, err error) {
	if value == nil {
		return nil, nil
	}

	slice, ok := value.([]any)
	if !ok {
		slice = []any{value}
	}

	ret = make([]string, len(slice))
	for i, v := range slice {
		if s, ok := v.(string); ok {
			ret[i] = s
		} else {
			bytes, _ := json.Marshal(v)
			ret[i] = string(bytes)
		}
	}
	return
}
