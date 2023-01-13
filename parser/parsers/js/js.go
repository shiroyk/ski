package js

import (
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/spf13/cast"
)

type Parser struct{}

const key string = "js"

func init() {
	parser.Register(key, new(Parser))
}

func (p *Parser) GetDesc() parser.Desc {
	desc := "Goja is an implementation of ECMAScript 5.1 in pure Go with emphasis on standard compliance and performance."
	return parser.Desc{
		Key:       key,
		Name:      "goja",
		Version:   "0.0.0",
		ShortDesc: desc,
		LongDesc:  desc,
		Url:       "https://github.com/dop251/goja",
	}
}

func (p *Parser) GetString(ctx *parser.Context, content any, arg string) (ret string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringE(str)
}

func (p *Parser) GetStrings(ctx *parser.Context, content any, arg string) (ret []string, err error) {
	str, err := runScript(ctx, content, arg)
	if err != nil {
		return
	}
	return cast.ToStringSliceE(str)
}

func (p *Parser) GetElement(ctx *parser.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

func (p *Parser) GetElements(ctx *parser.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

func runScript(ctx *parser.Context, content any, script string) (ret any, err error) {
	vm := js.CreateVMWithContext(ctx, content)

	result, err := vm.RunString(script)
	if err != nil {
		return
	}

	return js.UnWrapValue(result), err
}
