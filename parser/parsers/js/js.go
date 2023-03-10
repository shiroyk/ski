// Package js the js parser
package js

import (
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/parser"
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
func (p *Parser) GetString(ctx *parser.Context, content any, arg string) (ret string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringE(str)
}

// GetStrings gets the strings of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetStrings(ctx *parser.Context, content any, arg string) (ret []string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringSliceE(str)
}

// GetElement gets the element of the content with the given arguments.
// returns the string result.
func (p *Parser) GetElement(ctx *parser.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// returns the slice of string result.
func (p *Parser) GetElements(ctx *parser.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

func runScript(ctx *parser.Context, content any, script string) (any, error) {
	result, err := js.Run(ctx, common.Program{Code: script, Args: map[string]any{
		"content": content,
	}})
	if err != nil {
		return nil, err
	}

	return common.Unwrap(result)
}
