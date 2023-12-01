// Package js the js parser
package js

import (
	"hash/maphash"
	"sync"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/js/loader"
	"github.com/shiroyk/cloudcat/plugin"
)

// ESMParser the js parser with es module
type ESMParser struct {
	mu    *sync.Mutex
	cache map[uint64]goja.CyclicModuleRecord
	hash  *maphash.Hash
	load  func() loader.ModuleLoader
}

// NewESMParser returns a new ESMParser
func NewESMParser() *ESMParser {
	return &ESMParser{
		new(sync.Mutex),
		make(map[uint64]goja.CyclicModuleRecord),
		new(maphash.Hash),
		cloudcat.MustResolveLazy[loader.ModuleLoader](),
	}
}

// GetString gets the string of the content with the given arguments.
// returns the string result.
func (p *ESMParser) GetString(ctx *plugin.Context, content any, arg string) (ret string, err error) {
	v, err := p.run(ctx, content, arg)
	if err != nil {
		return "", err
	}
	return toString(v)
}

// GetStrings gets the strings of the content with the given arguments.
// returns the slice of string result.
func (p *ESMParser) GetStrings(ctx *plugin.Context, content any, arg string) (ret []string, err error) {
	v, err := p.run(ctx, content, arg)
	if err != nil {
		return nil, err
	}
	return toStrings(v)
}

// GetElement gets the element of the content with the given arguments.
// returns the string result.
func (p *ESMParser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// returns the slice of string result.
func (p *ESMParser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

// ClearCache clear the module cache
func (p *ESMParser) ClearCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	clear(p.cache)
}

func (p *ESMParser) run(ctx *plugin.Context, content any, script string) (any, error) {
	ctx.SetValue("content", content)

	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = p.hash.WriteString(script)
	hash := p.hash.Sum64()
	p.hash.Reset()

	mod, ok := p.cache[hash]
	if !ok {
		var err error
		mod, err = goja.ParseModule("", script, p.load().ResolveModule)
		if err != nil {
			return nil, err
		}
		p.cache[hash] = mod
	}

	result, err := js.RunModule(ctx, mod)
	if err != nil {
		return nil, err
	}

	return js.Unwrap(result)
}
