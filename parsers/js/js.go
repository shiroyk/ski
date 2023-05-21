// Package js the js parser
package js

import (
	"encoding/json"

	"github.com/shiroyk/cloudcat/core/js"
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
func (p *Parser) GetString(ctx *plugin.Context, content any, arg string) (ret string, err error) {
	return getString(ctx, content, arg)
}

// GetStrings gets the strings of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetStrings(ctx *plugin.Context, content any, arg string) (ret []string, err error) {
	return getStrings(ctx, content, arg)
}

// GetElement gets the element of the content with the given arguments.
// returns the string result.
func (p *Parser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return getString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return getStrings(ctx, content, arg)
}

func getString(ctx *plugin.Context, content any, script string) (ret string, err error) {
	result, err := js.Run(ctx, js.Program{Code: script, Args: map[string]any{
		"content": content,
	}})
	if err != nil {
		return ret, err
	}

	value, err := js.Unwrap(result)
	if err != nil {
		return ret, err
	}

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

func getStrings(ctx *plugin.Context, content any, script string) (ret []string, err error) {
	result, err := js.Run(ctx, js.Program{Code: script, Args: map[string]any{
		"content": content,
	}})
	if err != nil {
		return nil, err
	}

	value, err := js.Unwrap(result)
	if err != nil {
		return nil, err
	}
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
