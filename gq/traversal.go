package gq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/shiroyk/ski"
	"github.com/spf13/cast"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// contentToString converts content to strings,
// if node length is 1, return the first of the node strings.
func contentToString(content any, fn func(*goquery.Selection) (string, error)) (any, error) {
	switch c := content.(type) {
	case *goquery.Selection:
		list := make([]string, c.Length())
		c.EachWithBreak(func(i int, sel *goquery.Selection) bool {
			result, err := fn(sel)
			if err != nil {
				return false
			}
			list[i] = result
			return true
		})
		switch len(list) {
		case 0:
			return nil, nil
		case 1:
			return list[0], nil
		default:
			return list, nil
		}
	case *html.Node:
		return fn(goquery.NewDocumentFromNode(c).Selection)
	case []*html.Node:
		switch len(c) {
		case 0:
			return nil, nil
		case 1:
			return fn(goquery.NewDocumentFromNode(c[0]).Selection)
		}
		list := make([]string, len(c))
		for i, n := range c {
			result, err := fn(goquery.NewDocumentFromNode(n).Selection)
			if err != nil {
				return nil, err
			}
			list[i] = result
		}
		return list, nil
	case string, []string:
		return c, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	}
}

// gq_text returns the node text, if node list length is 1, return the first node text
func gq_text(_ ski.Arguments) (ski.Executor, error) { return text{}, nil }

type text struct{}

func (text) Exec(_ context.Context, arg any) (any, error) {
	return contentToString(arg, func(node *goquery.Selection) (string, error) {
		return strings.TrimSpace(node.Text()), nil
	})
}

// gq_attr gets the specified attribute's value for the first element in the
// Selection.
// The first argument is the name of the attribute, the second is the default value
func gq_attr(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("attr(name) must has name")
	}
	return attr{args.GetString(0), args.GetString(1)}, nil
}

type attr struct {
	name, defaultValue string
}

func (s attr) Exec(_ context.Context, arg any) (any, error) {
	return contentToString(arg, func(node *goquery.Selection) (string, error) {
		return node.AttrOr(s.name, s.defaultValue), nil
	})
}

// gq_href gets the href attribute's value, if URL is not absolute returns the absolute URL.
func gq_href(arg ski.Arguments) (ski.Executor, error) { return href(arg.GetString(0)), nil }

type href string

func (s href) Exec(ctx context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	href, exists := node.Attr("href")
	if !exists {
		return nil, errors.New("href attribute's value is not exist")
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href, nil
	}
	var base string

	if v := ctx.Value("baseURL"); v != nil {
		base = v.(string)
	} else if len(s) > 0 {
		base = string(s)
	}
	if len(base) > 0 {
		if !strings.HasPrefix(href, ".") {
			if !strings.HasPrefix(href, "/") {
				href = "/" + href
			}
			return strings.TrimSuffix(base, "/") + href, nil
		}
		hrefURL, err := url.Parse(href)
		if err != nil {
			return nil, err
		}
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, err
		}
		return baseURL.ResolveReference(hrefURL).String(), nil
	}
	return href, nil
}

// gq_html output the HTML text, the first argument is outer.
// If true returns the outer HTML rendering of the first item in
// the toSelection - that is, the HTML including the first element's
// tag and attributes, or gets the HTML contents of the first element
// in the set of matched elements. It includes text and comment nodes;
func gq_html(args ski.Arguments) (ski.Executor, error) {
	if len(args) > 0 {
		b, err := cast.ToBoolE(args.GetString(0))
		if err != nil {
			return nil, fmt.Errorf("html(outer) `outer` must bool type value: true/false")
		}
		return html_string(b), nil
	}
	return html_string(false), nil
}

type html_string bool

func (outer html_string) Exec(_ context.Context, arg any) (any, error) {
	return contentToString(arg, func(node *goquery.Selection) (string, error) {
		if outer {
			return goquery.OuterHtml(node)
		}
		return node.Html()
	})
}

// gq_prev gets the immediately preceding sibling of each element in the
// Selection.
// If present selector gets all preceding siblings of each element up to but not
// including the element matched by the selector.
func gq_prev(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return prev(nil), nil
	}
	ret, err := compileSelector(args)
	if err != nil {
		return nil, err
	}
	return prev(ret), nil
}

type prev cascadia.Selector

func (s prev) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return node.PrevUntilMatcher(cascadia.Selector(s)).Nodes, nil
	}
	return node.Prev().Nodes, nil
}

// gq_next gets the immediately following sibling of each element in the
// Selection.
// If present selector gets all following siblings of each element up to but not
// including the element matched by the selector.
func gq_next(args ski.Arguments) (ski.Executor, error) {
	if len(args) > 0 {
		ret, err := compileSelector(args)
		if err != nil {
			return nil, err
		}
		return next(ret), nil
	}
	return next(nil), nil
}

type next cascadia.Selector

func (s next) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return node.NextUntilMatcher(cascadia.Selector(s)).Nodes, nil
	}
	return node.Next().Nodes, nil
}

// gq_slice reduces the set of matched elements to a subset specified by a range
// of indices. The start index is 0-based and indicates the index of the first
// element to select. The end index is 0-based and indicates the index at which
// the elements stop being selected (the end index is not selected).
//
// If the end index is not specified reduces the set of matched elements to the one at the
// specified start index.
//
// The indices may be negative, in which case they represent an offset from the
// end of the toSelection.
func gq_slice(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, errors.New(`slice(start, end) must have at least one int argument`)
	}
	start, err := cast.ToIntE(args.GetString(0))
	if err != nil {
		return nil, err
	}
	if len(args) > 1 {
		end, err := cast.ToIntE(args.GetString(1))
		if err != nil {
			return nil, err
		}
		return slice{start, &end}, nil
	}
	return slice{start, nil}, nil
}

type slice struct {
	start int
	end   *int
}

func (s slice) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s.end == nil {
		return node.Eq(s.start).Nodes, nil
	}
	start := s.start
	end := *s.end
	if start < 0 {
		start += len(node.Nodes)
	}
	if start < 0 || start > len(node.Nodes) {
		return nil, nil
	}
	if end < 0 {
		end += len(node.Nodes)
	}
	if end < 0 || end > len(node.Nodes) {
		return nil, nil
	}
	return node.Nodes[start:end], nil
}

// gq_chunk return an array of elements split into groups the length of size
func gq_chunk(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, errors.New(`chunk(selector, size) must have at least one int argument`)
	}
	size := 1
	first := args.GetString(0)
	if isNumer(first) {
		return chunk{cast.ToInt(first), nil}, nil
	}
	sel, err := compileSelector(args)
	if err != nil {
		return nil, err
	}
	if len(args) > 1 {
		size, err = cast.ToIntE(args.GetString(1))
		if err != nil {
			return nil, err
		}
	}
	return chunk{size, sel}, nil
}

type chunk struct {
	size int
	sel  cascadia.Selector
}

func (s chunk) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	nodes := node.Nodes
	if s.sel != nil {
		nodes = node.FindMatcher(s.sel).Nodes
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	ret := make([]*html.Node, 0, len(nodes)/s.size)
	parent := nodes[0].Parent
	if parent == nil {
		parent = &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Div,
		}
	}
	for i := 0; i < len(nodes); i += s.size {
		p := &html.Node{
			Type:     parent.Type,
			DataAtom: parent.DataAtom,
			Data:     parent.Data,
			Attr:     parent.Attr,
		}
		end := min(s.size, len(nodes[i:]))
		for _, n := range nodes[i : i+end : i+end] {
			p.AppendChild(cloneNode(n))
		}
		ret = append(ret, p)
	}
	return ret, nil
}

// gq_child gets the child elements of each element in the Selection.
// If present the selector will return filtered by the specified selector.
func gq_child(args ski.Arguments) (ski.Executor, error) {
	if len(args) > 0 {
		ret, err := compileSelector(args)
		if err != nil {
			return nil, err
		}
		return child(ret), nil
	}
	return child(nil), nil
}

type child cascadia.Selector

func (s child) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return node.ChildrenMatcher(cascadia.Selector(s)).Nodes, nil
	}
	return node.Children().Nodes, nil
}

// gq_parent gets the parent of each element in the Selection.
// if present the selector will return filtered by a selector.
func gq_parent(args ski.Arguments) (ski.Executor, error) {
	if len(args) > 0 {
		ret, err := compileSelector(args)
		if err != nil {
			return nil, err
		}
		return parent(ret), nil
	}
	return parent(nil), nil
}

type parent cascadia.Selector

func (s parent) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return node.ParentMatcher(cascadia.Selector(s)).Nodes, nil
	}
	return node.Parent().Nodes, nil
}

// parents gets the ancestors of each element in the current Selection.
// if present the selector will return filtered by a selector.
func gq_parents(args ski.Arguments) (ski.Executor, error) {
	if len(args) > 0 {
		ret, err := compileSelector(args)
		if err != nil {
			return nil, err
		}
		return parents(ret), nil
	}
	return parents(nil), nil
}

type parents cascadia.Selector

func (s parents) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return node.ParentsMatcher(cascadia.Selector(s)).Nodes, nil
	}
	return node.Parents().Nodes, nil
}

// gq_closest gets the first element that matches the selector by testing the
// element itself and traversing up through its ancestors in the DOM tree.
func gq_closest(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("closest(selector) must has selector")
	}
	sel, err := compileSelector(args)
	if err != nil {
		return nil, err
	}
	return closest(sel), nil
}

type closest cascadia.Selector

func (s closest) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	return node.ClosestMatcher(cascadia.Selector(s)).Nodes, nil
}

// gq_not removes elements that match the given matcher.
// It returns nodes with the matching elements removed.
func gq_not(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("not(selector) must has selector")
	}
	sel, err := compileSelector(args)
	if err != nil {
		return nil, err
	}
	return not(sel), nil
}

type not cascadia.Selector

func (s not) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	return node.NotMatcher(cascadia.Selector(s)).Nodes, nil
}

// gq_has reduces the set of matched elements to those that have a descendant
// that matches the matcher.
func gq_has(args ski.Arguments) (ski.Executor, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("has(selector) must has selector")
	}
	sel, err := compileSelector(args)
	if err != nil {
		return nil, err
	}
	return has(sel), nil
}

type has cascadia.Selector

func (s has) Exec(_ context.Context, arg any) (any, error) {
	node, err := toSelection(arg)
	if err != nil {
		return nil, err
	}
	return node.HasMatcher(cascadia.Selector(s)).Nodes, nil
}
