// Package xpath the xpath executor
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

func init() {
	ski.Registers(ski.NewExecutors{
		"xpath":          xpath_value,
		"xpath.element":  xpath_element,
		"xpath.elements": xpath_elements,
	})
}

// xpath_value executes xpath selector and returns the result
// if length is 1, return the first of the result
func xpath_value(arg ski.Arguments) (ski.Executor, error) {
	ex, err := xpath.Compile(arg.GetString(0))
	if err != nil {
		return nil, err
	}
	return expr{ex, value}, nil
}

// xpath_element executes xpath selector and returns the first element
func xpath_element(arg ski.Arguments) (ski.Executor, error) {
	ex, err := xpath.Compile(arg.GetString(0))
	if err != nil {
		return nil, err
	}
	return expr{ex, element}, nil
}

// xpath_elements executes xpath selector and returns all elements
func xpath_elements(arg ski.Arguments) (ski.Executor, error) {
	ex, err := xpath.Compile(arg.GetString(0))
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
	return nodes, nil
}

func htmlNode(content any) (*html.Node, error) {
	switch data := content.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	case nil:
		return &html.Node{Type: html.DocumentNode}, nil
	case []*html.Node:
		root := &html.Node{Type: html.DocumentNode}
		if len(data) == 0 {
			return root, nil
		}
		for _, n := range data {
			root.AppendChild(n)
		}
		return root, nil
	case *html.Node:
		return data, nil
	case []string:
		return html.Parse(strings.NewReader(strings.Join(data, "")))
	case fmt.Stringer:
		return html.Parse(strings.NewReader(data.String()))
	case string:
		return html.Parse(strings.NewReader(data))
	}
}
