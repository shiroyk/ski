// Package xpath the xpath parser
package xpath

import (
	"context"
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/antchfx/xpath"
	"github.com/shiroyk/ski"
	"golang.org/x/net/html"
)

// Parser the xpath parser
type Parser struct{}

func init() {
	ski.Register("xpath", new(Parser))
}

func (p Parser) Value(arg string) (ski.Executor, error) {
	ex, err := xpath.Compile(arg)
	if err != nil {
		return nil, err
	}
	return expr{ex, value}, nil
}

func (p Parser) Element(arg string) (ski.Executor, error) {
	ex, err := xpath.Compile(arg)
	if err != nil {
		return nil, err
	}
	return expr{ex, element}, nil
}
func (p Parser) Elements(arg string) (ski.Executor, error) {
	ex, err := xpath.Compile(arg)
	if err != nil {
		return nil, err
	}
	return expr{ex, elements}, nil
}

type expr struct {
	*xpath.Expr
	ret func([]*html.Node) (any, error)
}

func (e expr) Exec(_ context.Context, arg any) (any, error) {
	node, err := htmlNode(arg)
	if err != nil {
		return nil, err
	}
	return e.ret(htmlquery.QuerySelectorAll(node, e.Expr))
}

func value(nodes []*html.Node) (any, error) {
	switch len(nodes) {
	case 0:
		return nil, nil
	case 1:
		return htmlquery.InnerText(nodes[0]), nil
	default:
		str := make([]string, len(nodes))
		for i, node := range nodes {
			str[i] = htmlquery.InnerText(node)
		}
		return str, nil
	}
}

func element(nodes []*html.Node) (any, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

func elements(nodes []*html.Node) (any, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	ret := make([]any, len(nodes))
	for i, node := range nodes {
		ret[i] = node
	}

	return ret, nil
}

func htmlNode(content any) (node *html.Node, err error) {
	switch data := content.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	case nil:
		return nil, nil
	case *html.Node:
		return data, nil
	case []string:
		return html.Parse(strings.NewReader(strings.Join(data, "\n")))
	case string:
		return html.Parse(strings.NewReader(data))
	}
}
