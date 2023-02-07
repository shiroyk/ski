package parser

import (
	"github.com/shiroyk/cloudcat/ext"
)

type Parser interface {
	GetString(*Context, any, string) (string, error)
	GetStrings(*Context, any, string) ([]string, error)
	GetElement(*Context, any, string) (string, error)
	GetElements(*Context, any, string) ([]string, error)
}

func Register(key string, parser Parser) {
	ext.Register(key, ext.ParserExtension, parser)
}

func GetParser(key string) (Parser, bool) {
	parsers := ext.Get(ext.ParserExtension)
	if parser, ok := parsers[key]; ok {
		return parser.Module.(Parser), true
	}
	return nil, false
}
