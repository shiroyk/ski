package parser

import (
	"github.com/shiroyk/cloudcat/ext"
)

// Parser the content parser
type Parser interface {
	GetString(*Context, any, string) (string, error)
	GetStrings(*Context, any, string) ([]string, error)
	GetElement(*Context, any, string) (string, error)
	GetElements(*Context, any, string) ([]string, error)
}

// Register registers the parser with the given key parser
func Register(key string, parser Parser) {
	ext.Register(key, ext.ParserExtension, parser)
}

// GetParser returns a Parser with the given key
func GetParser(key string) (Parser, bool) {
	parsers := ext.Get(ext.ParserExtension)
	if parser, ok := parsers[key]; ok {
		return parser.Module.(Parser), true
	}
	return nil, false
}
