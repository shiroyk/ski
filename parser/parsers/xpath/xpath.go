package xpath

import (
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/utils"
	"golang.org/x/net/html"
)

// Parser the xpath parser
type Parser struct{}

const key string = "xpath"

func init() {
	parser.Register(key, new(Parser))
}

func (p Parser) GetString(ctx *parser.Context, content any, arg string) (string, error) {
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
		str.WriteString(ctx.Config().Separator)
		str.WriteString(htmlquery.InnerText(node))
	}

	return str.String(), nil
}

func (p Parser) GetStrings(_ *parser.Context, content any, arg string) ([]string, error) {
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

func (p Parser) GetElement(_ *parser.Context, content any, arg string) (string, error) {
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

func (p Parser) GetElements(_ *parser.Context, content any, arg string) ([]string, error) {
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
	switch data := utils.FromPtr(content).(type) {
	default:
		return nil, fmt.Errorf("unexpected content type %T", data)
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
