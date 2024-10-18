// Package gq the goquery executor
package gq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unsafe"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/shiroyk/ski"
	"golang.org/x/net/html"
)

func init() {
	ski.Registers(ski.NewExecutors{
		"gq":          gq,
		"gq.element":  gq_element,
		"gq.elements": gq_elements,
		"gq.attr":     gq_attr,
		"gq.text":     gq_text,
		"gq.html":     gq_html,
		"gq.href":     gq_href,
		"gq.next":     gq_next,
		"gq.prev":     gq_prev,
		"gq.chunk":    gq_chunk,
		"gq.child":    gq_child,
		"gq.slice":    gq_slice,
		"gq.parent":   gq_parent,
		"gq.parents":  gq_parents,
	})
}

// gq executes selector and returns the node text,
// if node list length is 1, return the first node text
func gq(args ski.Arguments) (ski.Executor, error) {
	pipe, err := compile(args)
	if err != nil {
		return nil, err
	}
	return append(pipe, value{}), nil
}

// gq_element executes goquery selector and returns the first node
func gq_element(args ski.Arguments) (ski.Executor, error) {
	pipe, err := compile(args)
	if err != nil {
		return nil, err
	}
	return append(pipe, element{}), nil
}

// gq_elements executes goquery selector and returns all nodes
func gq_elements(args ski.Arguments) (ski.Executor, error) {
	pipe, err := compile(args)
	if err != nil {
		return nil, err
	}
	return append(pipe, elements{}), nil
}

// compile the gq function
func compile(args ski.Arguments) (ski.Pipe, error) {
	funcs := strings.Split(args.GetString(0), "->")

	if len(funcs) == 1 {
		v, err := cascadia.Compile(funcs[0])
		return ski.Pipe{selector(v)}, err
	}

	pipe := make(ski.Pipe, 0, len(funcs))

	if s := strings.TrimSpace(funcs[0]); len(s) > 0 {
		sel, err := cascadia.Compile(s)
		if err != nil {
			return nil, err
		}
		pipe = append(pipe, selector(sel))
	}

	for _, function := range funcs[1:] {
		function = strings.TrimSpace(function)
		if function == "" {
			continue
		}
		name, args, err := parseFuncArguments(function)
		if err != nil {
			return nil, err
		}

		if strings.IndexByte(name, '.') == -1 {
			name = fmt.Sprintf("gq.%s", name)
		}
		ne, ok := ski.GetExecutor(name)
		if !ok {
			return nil, fmt.Errorf("function %s not exists", name)
		}
		e, err := ne(args)
		if err != nil {
			return nil, fmt.Errorf("compile function %s failed: %s", name, err)
		}
		pipe = append(pipe, e)
	}

	return pipe, nil
}

// compileSelector compile the selector
func compileSelector(args ski.Arguments) (ret cascadia.Selector, err error) {
	selector := args.GetString(0)
	if len(selector) == 0 {
		return ret, errors.New("missing selector argument")
	}
	return cascadia.Compile(selector)
}

type selector cascadia.Selector

func (s selector) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	return node.FindMatcher(cascadia.Selector(s)).Nodes, nil
}

type value = text

type element struct{}

func (element) Exec(_ context.Context, node any) (any, error) {
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

type elements struct{}

func (elements) Exec(_ context.Context, node any) (any, error) {
	switch t := node.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", node)
	case string, []string, *html.Node, []*html.Node, nil:
		return t, nil
	case *goquery.Selection:
		return t.Nodes, nil
	}
}

func cloneNode(n *html.Node) *html.Node {
	m := &html.Node{
		Type:       n.Type,
		DataAtom:   n.DataAtom,
		Data:       n.Data,
		Attr:       make([]html.Attribute, len(n.Attr)),
		FirstChild: n.FirstChild,
		LastChild:  n.LastChild,
	}
	copy(m.Attr, n.Attr)
	return m
}

// toSelection converts content to goquery.Selection
func toSelection(content any) (*goquery.Selection, error) {
	switch data := content.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	case nil:
		return new(goquery.Selection), nil
	case *html.Node:
		return goquery.NewDocumentFromNode(data).Selection, nil
	case []*html.Node:
		if len(data) == 0 {
			return new(goquery.Selection), nil
		}
		root := data[0].Parent
		if root == nil {
			root = &html.Node{Type: html.DocumentNode}
			for _, n := range data {
				root.AppendChild(n)
			}
		}
		d := &document{nil, nil, root}
		d.Selection = &selection{data, d, nil}
		doc := (*goquery.Document)(unsafe.Pointer(d))
		return doc.Selection, nil
	case []string:
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(strings.Join(data, "")))
		if err != nil {
			return nil, err
		}
		return doc.Selection, nil
	case fmt.Stringer:
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(data.String()))
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

type document struct {
	Selection *selection
	Url       *url.URL
	rootNode  *html.Node
}

type selection struct {
	Nodes    []*html.Node
	document *document
	prevSel  *selection
}
