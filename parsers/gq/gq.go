// Package gq the goquery parser
package gq

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/shiroyk/ski"
	"golang.org/x/net/html"
)

// parser the goquery parser
type parser struct{ funcs FuncMap }

// NewParser creates a new goquery parser with the given FuncMap.
func NewParser(m FuncMap) ski.ElementParser {
	funcs := maps.Clone(builtins())
	maps.Copy(funcs, m)
	return &parser{funcs}
}

func init() {
	ski.Register("gq", NewParser(nil))
}

func (p *parser) Value(arg string) (ski.Executor, error) {
	ret, err := p.compile(arg)
	if err != nil {
		return nil, err
	}
	ret.calls = append(ret.calls, call{fn: value})
	return ret, nil
}

func (p *parser) Element(arg string) (ski.Executor, error) {
	ret, err := p.compile(arg)
	if err != nil {
		return nil, err
	}
	ret.calls = append(ret.calls, call{fn: element})
	return ret, nil
}

func (p *parser) Elements(arg string) (ski.Executor, error) {
	ret, err := p.compile(arg)
	if err != nil {
		return nil, err
	}
	ret.calls = append(ret.calls, call{fn: elements})
	return ret, nil
}

func (p *parser) compile(raw string) (ret matcher, err error) {
	funcs := strings.Split(raw, "->")
	if len(funcs) == 1 {
		ret.Matcher, err = cascadia.Compile(funcs[0])
		return
	}
	selector := strings.TrimSpace(funcs[0])
	if len(selector) == 0 {
		ret.Matcher = new(emptyMatcher)
	} else {
		ret.Matcher, err = cascadia.Compile(selector)
		if err != nil {
			return
		}
	}

	ret.calls = make([]call, 0, len(funcs)-1)

	for _, function := range funcs[1:] {
		function = strings.TrimSpace(function)
		if function == "" {
			continue
		}
		name, args, err := parseFuncArguments(function)
		if err != nil {
			return ret, err
		}
		fn, ok := p.funcs[name]
		if !ok {
			return ret, fmt.Errorf("function %s not exists", name)
		}
		ret.calls = append(ret.calls, call{fn, args})
	}

	return
}

type call struct {
	fn   Func
	args []string
}

type matcher struct {
	goquery.Matcher
	calls []call
}

func (f matcher) Exec(ctx context.Context, arg any) (any, error) {
	nodes, err := selection(arg)
	if err != nil {
		return nil, err
	}

	var node any = nodes.FindMatcher(f)

	for _, c := range f.calls {
		node, err = c.fn(ctx, node, c.args...)
		if err != nil || node == nil {
			return nil, err
		}
	}

	return node, nil
}

func value(ctx context.Context, node any, _ ...string) (any, error) {
	v, err := Text(ctx, node)
	if node == nil || err != nil {
		return nil, err
	}
	return v, nil
}

func element(_ context.Context, node any, _ ...string) (any, error) {
	switch t := node.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", node)
	case string, []string, *html.Node, nil:
		return t, nil
	case *goquery.Selection:
		if len(t.Nodes) == 0 {
			return nil, nil
		}
		return t.Nodes[0], nil
	case []*html.Node:
		if len(t) == 0 {
			return nil, nil
		}
		return t[0], nil
	}
}

func elements(_ context.Context, node any, _ ...string) (any, error) {
	switch t := node.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", node)
	case string, []string, *html.Node, nil:
		return t, nil
	case *goquery.Selection:
		ele := make([]any, t.Length())
		for i, n := range t.Nodes {
			ele[i] = n
		}
		return ele, nil
	case []*html.Node:
		ele := make([]any, len(t))
		for i, n := range t {
			ele[i] = n
		}
		return ele, nil
	}
}

// selection converts content to goquery.Selection
func selection(content any) (*goquery.Selection, error) {
	switch data := content.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	case nil:
		return new(goquery.Selection), nil
	case *html.Node:
		return goquery.NewDocumentFromNode(data).Selection, nil
	case []any:
		if len(data) == 0 {
			return nil, nil
		}
		root := &html.Node{Type: html.DocumentNode}
		doc := goquery.NewDocumentFromNode(root)
		doc.Selection.Nodes = make([]*html.Node, len(data))
		for i, v := range data {
			n, ok := v.(*html.Node)
			if !ok {
				return nil, fmt.Errorf("expected type *html.Node, but got %T", v)
			}
			n.Parent = nil
			n.PrevSibling = nil
			n.NextSibling = nil
			root.AppendChild(n)
			doc.Selection.Nodes[i] = n
		}
		return doc.Selection, nil
	case []string:
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(strings.Join(data, "\n")))
		if err != nil {
			return nil, err
		}
		return doc.Selection, nil
	case string:
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
		if err != nil {
			return nil, err
		}
		return doc.Selection, nil
	}
}

type emptyMatcher struct{}

func (emptyMatcher) Match(*html.Node) bool { return true }

func (emptyMatcher) MatchAll(node *html.Node) []*html.Node { return []*html.Node{node} }

func (emptyMatcher) Filter(nodes []*html.Node) []*html.Node { return nodes }
