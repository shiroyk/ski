package ski

import (
	"maps"
	"sync"
)

// Parser compile the selector return the Executor
type Parser interface {
	// Value get the value of the content with the given argument.
	//
	// content := `<ul><li>1</li><li>2</li></ul>`
	// e, _ := Value("ul li")
	// e.Exec(ctx, content) // "1\n2"
	Value(string) (Executor, error)
}

// ElementParser compile the selector return the Executor
type ElementParser interface {
	Parser

	// Element get the element of the content with the given argument.
	//
	// content := `<ul><li>1</li><li>2</li></ul>`
	// e, _ := Element("ul li")
	// e.Exec(ctx, content) // "<li>1</li>\n<li>2</li>"
	Element(string) (Executor, error)

	// Elements get the elements of the content with the given argument.
	//
	// content := `<ul><li>1</li><li>2</li></ul>`
	// e, _ := Elements("ul li")
	// e.Exec(ctx, content) // []string{"<li>1</li>", "<li>2</li>"}
	Elements(string) (Executor, error)
}

// Register registers the Parser with the given key Parser
func Register(key string, parser Parser) {
	parsers.Lock()
	defer parsers.Unlock()
	parsers.registry[key] = parser
}

// GetParser returns a Parser with the given key
func GetParser(key string) (Parser, bool) {
	parsers.RLock()
	defer parsers.RUnlock()
	parser, ok := parsers.registry[key]
	return parser, ok
}

func RemoveParser(key string) {
	parsers.Lock()
	defer parsers.Unlock()
	delete(parsers.registry, key)
}

func AllParser() map[string]Parser {
	parsers.RLock()
	defer parsers.RUnlock()
	return maps.Clone(parsers.registry)
}

var parsers = struct {
	sync.RWMutex
	registry map[string]Parser
}{
	registry: make(map[string]Parser),
}
