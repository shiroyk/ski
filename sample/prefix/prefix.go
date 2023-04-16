package main

import (
	"fmt"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
)

type Parser struct{}

func init() {
	parser.Register("prefix", new(Parser))
}

func (p Parser) GetString(_ *plugin.Context, content any, arg string) (string, error) {
	if str, ok := content.(string); ok {
		return arg + str, nil
	}
	return "", fmt.Errorf("content must be a string")
}

func (p Parser) GetStrings(_ *plugin.Context, content any, arg string) ([]string, error) {
	if str, ok := content.([]string); ok {
		for i := range str {
			str[i] = arg + str[i]
		}
	}
	return nil, fmt.Errorf("content must be a string slice")
}

func (p Parser) GetElement(_ *plugin.Context, content any, arg string) (string, error) {
	return p.GetString(nil, content, arg)
}

func (p Parser) GetElements(_ *plugin.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(nil, content, arg)
}
