// Package xpath the xpath parser
package xpath

import (
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/spf13/cast"
	"golang.org/x/net/html"
)

// Parser the xpath parser
type Parser struct{}

const key string = "xpath"

func init() {
	parser.Register(key, new(Parser))
}

// GetString gets the string of the content with the given arguments.
//
// content := `<ul><li>1</li><li>2</li></ul>`
// GetString(ctx, content, "//li/text()") returns "1\n2"
func (p Parser) GetString(_ *plugin.Context, content any, arg string) (string, error) {
	nodes, err := getHTMLNode(content, arg)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", nil
	}

	str := strings.Builder{}
	str.WriteString(htmlquery.InnerText(nodes[0]))
	for _, node := range nodes[1:] {
		str.WriteString("\n")
		str.WriteString(htmlquery.InnerText(node))
	}

	return str.String(), nil
}

// GetStrings gets the strings of the content with the given arguments.
//
// content := `<ul><li>1</li><li>2</li></ul>`
// GetStrings(ctx, content, "//li/text()") returns []string{"1", "2"}
func (p Parser) GetStrings(_ *plugin.Context, content any, arg string) ([]string, error) {
	nodes, err := getHTMLNode(content, arg)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	result := make([]string, len(nodes))
	for i, node := range nodes {
		result[i] = htmlquery.InnerText(node)
	}

	return result, err
}

// GetElement gets the element of the content with the given arguments.
//
// content := `<ul><li>1</li><li>2</li></ul>`
// GetStrings(ctx, content, "//li..") returns "<li>1</li>\n<li>2</li>"
func (p Parser) GetElement(_ *plugin.Context, content any, arg string) (string, error) {
	nodes, err := getHTMLNode(content, arg)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", nil
	}

	str := strings.Builder{}
	str.WriteString(htmlquery.OutputHTML(nodes[0], true))
	for _, node := range nodes[1:] {
		str.WriteString("\n")
		str.WriteString(htmlquery.OutputHTML(node, true))
	}

	return str.String(), nil
}

// GetElements gets the elements of the content with the given arguments.
//
// content := `<ul><li>1</li><li>2</li></ul>`
// GetStrings(ctx, content, "//li..") returns []string{"<li>1</li>", "<li>2</li>"}
func (p Parser) GetElements(_ *plugin.Context, content any, arg string) ([]string, error) {
	nodes, err := getHTMLNode(content, arg)
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	str := make([]string, len(nodes))
	for i, node := range nodes {
		str[i] = htmlquery.OutputHTML(node, true)
	}

	return str, nil
}

func getHTMLNode(content any, arg string) ([]*html.Node, error) {
	var err error
	var node *html.Node
	switch data := content.(type) {
	default:
		str, err := cast.ToStringE(content)
		if err != nil {
			return nil, err
		}
		node, err = html.Parse(strings.NewReader(str))
		if err != nil {
			return nil, err
		}
	case nil:
		return nil, nil
	case []string:
		node, err = html.Parse(strings.NewReader(strings.Join(data, "\n")))
		if err != nil {
			return nil, err
		}
	case string:
		node, err = html.Parse(strings.NewReader(data))
		if err != nil {
			return nil, err
		}
	}

	htmlNode, err := htmlquery.QueryAll(node, arg)
	if err != nil {
		return nil, err
	}

	return htmlNode, nil
}
