package html

import (
	"reflect"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	TypeNode  = reflect.TypeOf((*html.Node)(nil))
	TypeNodes = reflect.TypeOf(([]*html.Node)(nil))
)

var body = &html.Node{
	Type:     html.ElementNode,
	Data:     "body",
	DataAtom: atom.Body,
}

// Parse as html or fragment
func Parse(str string) (*html.Node, error) {
	reader := strings.NewReader(str)
	z := html.NewTokenizer(reader)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break
		case html.TextToken, html.CommentToken:
			continue
		case html.StartTagToken:
			_, _ = reader.Seek(0, 0)
			name, _ := z.TagName()
			if string(name) == `html` {
				return html.Parse(reader)
			}
			nodes, err := html.ParseFragment(reader, body)
			if err != nil {
				return nil, err
			}
			return MergeNode(nodes), nil
		default:
			_, _ = reader.Seek(0, 0)
			return html.Parse(reader)
		}
	}
}

// CloneNode deep clone the node
func CloneNode(n *html.Node) *html.Node {
	nn := &html.Node{
		Type:     n.Type,
		DataAtom: n.DataAtom,
		Data:     n.Data,
		Attr:     make([]html.Attribute, len(n.Attr)),
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		nn.AppendChild(CloneNode(c))
	}
	copy(nn.Attr, n.Attr)
	return nn
}

// MergeNode wrap the nodes with html.ElementNode
func MergeNode(nodes []*html.Node) *html.Node {
	if len(nodes) == 0 {
		return &html.Node{Type: html.DocumentNode}
	}
	root := nodes[0].Parent
	if root == nil {
		root = &html.Node{Type: html.DocumentNode}
		for _, n := range nodes {
			root.AppendChild(n)
		}
	} else {
		root = &html.Node{
			Type:     root.Type,
			DataAtom: root.DataAtom,
			Data:     root.Data,
			Attr:     root.Attr,
		}
		for _, n := range nodes {
			root.AppendChild(CloneNode(n))
		}
	}
	return root
}
