package js

import (
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/schema/parsers"
	"github.com/spf13/cast"
)

// Parser the js schema
type Parser struct{}

const key string = "js"

func init() {
	parsers.Register(key, new(Parser))
}

func (p *Parser) GetString(ctx *parsers.Context, content any, arg string) (ret string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringE(str)
}

func (p *Parser) GetStrings(ctx *parsers.Context, content any, arg string) (ret []string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringSliceE(str)
}

func (p *Parser) GetElement(ctx *parsers.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

func (p *Parser) GetElements(ctx *parsers.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

func runScript(ctx *parsers.Context, content any, script string) (any, error) {
	result, err := js.Run(ctx, common.Program{Code: script, Args: map[string]any{
		"content": content,
	}})
	if err != nil {
		return nil, err
	}

	return common.Unwrap(result)
}
