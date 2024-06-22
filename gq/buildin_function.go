package gq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/shiroyk/ski"
	"github.com/spf13/cast"
	"golang.org/x/net/html"
)

type (
	// Func is the type of gq parse function.
	Func func(ctx context.Context, content any, args ...string) (any, error)
	// FuncMap is the type of the map defining the mapping from names to functions.
	FuncMap map[string]Func
)

func builtins() FuncMap {
	return FuncMap{
		"zip":     Zip,
		"attr":    Attr,
		"href":    Href,
		"html":    Html,
		"prev":    Prev,
		"text":    Text,
		"next":    Next,
		"slice":   Slice,
		"child":   Child,
		"parent":  Parent,
		"parents": Parents,
		"prefix":  Prefix,
		"suffix":  Suffix,
	}
}

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
			return ski.NewIterator(list), nil
		}
	case *html.Node:
		return fn(goquery.NewDocumentFromNode(c).Selection)
	case string, ski.Iterator:
		return c, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", content)
	}
}

// Text gets the combined text contents of each element in the set of matched
// elements, including their descendants.
func Text(_ context.Context, content any, _ ...string) (any, error) {
	return contentToString(content, func(node *goquery.Selection) (string, error) {
		return strings.TrimSpace(node.Text()), nil
	})
}

// Attr gets the specified attribute's value for the first element in the
// Selection.
// The first argument is the name of the attribute, the second is the default value
func Attr(_ context.Context, content any, args ...string) (any, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("attr(name) must has name")
	}

	name := args[0]
	defaultValue := ""
	if len(args) > 1 {
		defaultValue = args[1]
	}

	return contentToString(content, func(node *goquery.Selection) (string, error) {
		return node.AttrOr(name, defaultValue), nil
	})
}

// Href gets the href attribute's value, if URL is not absolute returns the absolute URL.
func Href(ctx context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
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
		} else if len(args) > 0 {
			base = args[0]
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

	return nil, fmt.Errorf("href: unexpected content type %T", content)
}

// Html the first argument is outer.
// If true returns the outer HTML rendering of the first item in
// the selection - that is, the HTML including the first element's
// tag and attributes, or gets the HTML contents of the first element
// in the set of matched elements. It includes text and comment nodes;
func Html(_ context.Context, content any, args ...string) (any, error) { //nolint
	var err error
	var outer bool

	if len(args) > 0 {
		outer, err = cast.ToBoolE(args[0])
		if err != nil {
			return nil, fmt.Errorf("html(outer) `outer` must bool type value: true/false")
		}
	}

	return contentToString(content, func(node *goquery.Selection) (string, error) {
		var str string

		if outer {
			str, err = goquery.OuterHtml(node)
			if err != nil {
				return "", err
			}
		} else {
			str, err = node.Html()
			if err != nil {
				return "", err
			}
		}

		return str, nil
	})
}

// Prev gets the immediately preceding sibling of each element in the
// Selection.
// If present selector gets all preceding siblings of each element up to but not
// including the element matched by the selector.
func Prev(_ context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.PrevUntil(args[0]), nil
		}
		return node.Prev(), nil
	}

	return nil, fmt.Errorf("prev: unexpected content type %T", content)
}

// Next gets the immediately following sibling of each element in the
// Selection.
// If present selector gets all following siblings of each element up to but not
// including the element matched by the selector.
func Next(_ context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.NextUntil(args[0]), nil
		}
		return node.Next(), nil
	}

	return nil, fmt.Errorf("next: unexpected type %T", content)
}

// Slice reduces the set of matched elements to a subset specified by a range
// of indices. The start index is 0-based and indicates the index of the first
// element to select. The end index is 0-based and indicates the index at which
// the elements stop being selected (the end index is not selected).
//
// If the end index is not specified reduces the set of matched elements to the one at the
// specified start index.
//
// The indices may be negative, in which case they represent an offset from the
// end of the selection.
func Slice(_ context.Context, content any, args ...string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("slice(start, end) must have at least one int argument")
	}

	if node, ok := content.(*goquery.Selection); ok {
		var err error
		var start, end int
		if start, err = cast.ToIntE(args[0]); err != nil {
			return nil, err
		}

		if len(args) > 1 {
			if end, err = cast.ToIntE(args[1]); err != nil {
				return nil, err
			}
			return node.Slice(start, end), nil
		}
		return node.Eq(start), nil
	}

	return nil, fmt.Errorf("slice: unexpected type %T", content)
}

// Child gets the child elements of each element in the Selection.
// If present the selector will return filtered by the specified selector.
func Child(_ context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ChildrenFiltered(args[0]), nil
		}
		return node.Children(), nil
	}

	return nil, fmt.Errorf("child: unexpected type %T", content)
}

// Parent gets the parent of each element in the Selection.
// if present the selector will return filtered by a selector.
func Parent(_ context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ParentFiltered(args[0]), nil
		}
		return node.Parent(), nil
	}

	return nil, fmt.Errorf("parent: unexpected type %T", content)
}

// Parents gets the ancestors of each element in the current Selection.
// if present the selector will return filtered by a selector.
func Parents(_ context.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok { //nolint:nestif
		if len(args) > 0 {
			if len(args) > 1 {
				until, err := cast.ToBoolE(args[1])
				if err != nil {
					return nil, fmt.Errorf("parents(selector, until) `until` must bool type value: true/false")
				}
				if until {
					return node.ParentsUntil(args[0]), nil
				}
			}
			return node.ParentsFiltered(args[0]), nil
		}
		return node.Parents(), nil
	}

	return nil, fmt.Errorf("parents: unexpected type %T", content)
}

// Zip returns an element array of first selector element length,
// the first of which contains the first elements of the given selector,
// the second of which contains the second elements of the given selector, and so on.
func Zip(_ context.Context, content any, args ...string) (any, error) {
	sel, ok := content.(*goquery.Selection)
	if !ok {
		return nil, fmt.Errorf("zip: unexpected type %T", content)
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("zip(selector) must have at least one string argument")
	}

	first := sel.Find(args[0])
	length := first.Length()
	zip := make([]string, 0, length*len(args))
	first.Each(func(i int, s *goquery.Selection) {
		html, _ := goquery.OuterHtml(s)
		zip = append(zip, html)
	})

	for _, arg := range args[1:] {
		sel.Find(arg).Each(func(i int, s *goquery.Selection) {
			html, _ := goquery.OuterHtml(s)
			zip = append(zip, html)
		})
	}

	ret := make([]string, 0, length)
	for i := 0; i < length; i++ {
		var s string
		for j := 0; j < len(args); j++ {
			s += zip[i+j*length]
		}
		ret = append(ret, s)
	}

	return ret, nil
}

func Prefix(_ context.Context, content any, args ...string) (ret any, err error) {
	if len(args) == 0 {
		return content, nil
	}
	switch src := content.(type) {
	case string:
		return args[0] + src, nil
	case ski.Iterator:
		ret := make([]string, src.Len())
		for i := 0; i < src.Len(); i++ {
			if s, ok := src.At(i).(string); ok {
				ret[i] = args[0] + s
			} else {
				return nil, fmt.Errorf("prefix: unexpected type %T", src.At(i))
			}
		}
		return ret, nil
	case *goquery.Selection:
		return args[0] + src.Text(), nil
	default:
		return content, nil
	}
}

func Suffix(_ context.Context, content any, args ...string) (any, error) {
	if len(args) == 0 {
		return content, nil
	}
	switch src := content.(type) {
	case string:
		return src + args[0], nil
	case ski.Iterator:
		ret := make([]string, src.Len())
		for i := 0; i < src.Len(); i++ {
			if s, ok := src.At(i).(string); ok {
				ret[i] = s + args[0]
			} else {
				return nil, fmt.Errorf("suffix: unexpected type %T", src.At(i))
			}
		}
		return ret, nil
	case *goquery.Selection:
		return src.Text() + args[0], nil
	default:
		return content, nil
	}
}
