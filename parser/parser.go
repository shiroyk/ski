package parser

import (
	"github.com/shiroyk/cloudcat/internal/ext"
)

// Parser the content schema
type Parser interface {
	// GetString gets the string of the content with the given arguments
	GetString(*Context, any, string) (string, error)
	// GetStrings gets the string of the content with the given arguments
	GetStrings(*Context, any, string) ([]string, error)
	// GetElement gets the string of the content with the given arguments
	GetElement(*Context, any, string) (string, error)
	// GetElements gets the string of the content with the given arguments
	GetElements(*Context, any, string) ([]string, error)
}

// Register registers the Parser with the given key Parser
func Register(key string, parser Parser) {
	ext.Register(key, ext.ParserExtension, parser)
}

// GetParser returns a Parser with the given key
func GetParser(key string) (Parser, bool) {
	parsers := ext.Get(ext.ParserExtension)
	if p, ok := parsers[key]; ok {
		return p.Module.(Parser), true
	}
	return nil, false
}
