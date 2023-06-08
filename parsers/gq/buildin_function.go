package gq

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cast"
)

type (
	// GFunc is the type of gq parse function.
	GFunc func(ctx *plugin.Context, content any, args ...string) (any, error)
	// FuncMap is the type of the map defining the mapping from names to functions.
	FuncMap map[string]GFunc
)

func builtins() FuncMap {
	return FuncMap{
		"get":     Get,
		"set":     Set,
		"attr":    Attr,
		"href":    Href,
		"html":    Html,
		"join":    Join,
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
	if node, ok := content.(*goquery.Selection); ok {
		list := make([]string, node.Length())
		node.EachWithBreak(func(i int, sel *goquery.Selection) bool {
			result, err := fn(sel)
			if err != nil {
				return false
			}
			list[i] = result
			return true
		})
		if len(list) == 1 {
			return list[0], nil
		}
		return list, nil
	}
	return nil, fmt.Errorf("unexpected type %T", content)
}

// Get returns the value associated with this context for key, or nil
// if no value is associated with key.
func Get(ctx *plugin.Context, _ any, args ...string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("get function must has one argment")
	}

	key, err := cast.ToStringE(args[0])
	if err != nil {
		return nil, err
	}

	return ctx.Value(key), nil
}

// Set value associated with key is val.
// The first argument is the key, and the second argument is value.
// if the value is present will store the content.
func Set(ctx *plugin.Context, content any, args ...string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("set function must has least one argment")
	}

	key, err := cast.ToStringE(args[0])
	if err != nil {
		return nil, err
	}

	if len(args) > 1 {
		ctx.SetValue(key, args[1])
	} else {
		ctx.SetValue(key, content)
	}

	return content, nil
}

// Text gets the combined text contents of each element in the set of matched
// elements, including their descendants.
func Text(_ *plugin.Context, content any, _ ...string) (any, error) {
	return contentToString(content, func(node *goquery.Selection) (string, error) {
		return strings.TrimSpace(node.Text()), nil
	})
}

// Join the text with the separator, if not present separator uses the default separator ", ".
func Join(ctx *plugin.Context, content any, args ...string) (any, error) {
	if str, ok := content.(string); ok {
		return str, nil
	}

	if node, ok := content.(*goquery.Selection); ok {
		text, err := Text(ctx, node)
		if err != nil {
			return nil, err
		}
		if str, ok := text.(string); ok {
			return str, nil
		}
		content = text
	}

	list, err := cast.ToStringSliceE(content)
	if err != nil {
		return nil, err
	}

	sep := ", "
	if len(args) > 0 {
		sep = args[0]
	}

	return strings.Join(list, sep), nil
}

// Attr gets the specified attribute's value for the first element in the
// Selection.
// The first argument is the name of the attribute, the second is the default value
func Attr(_ *plugin.Context, content any, args ...string) (any, error) {
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
func Href(ctx *plugin.Context, content any, _ ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		hrefURL, err := url.Parse(node.AttrOr("href", ""))
		if err != nil {
			return nil, err
		}

		baseURL, err := url.Parse(ctx.BaseURL())
		if err != nil {
			return nil, err
		}
		return baseURL.ResolveReference(hrefURL), nil
	}

	return nil, fmt.Errorf("unexpected content type %T", content)
}

// Html the first argument is outer.
// If true returns the outer HTML rendering of the first item in
// the selection - that is, the HTML including the first element's
// tag and attributes, or gets the HTML contents of the first element
// in the set of matched elements. It includes text and comment nodes;
func Html(_ *plugin.Context, content any, args ...string) (any, error) { //nolint
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
func Prev(_ *plugin.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.PrevUntil(args[0]), nil
		}
		return node.Prev(), nil
	}

	return nil, fmt.Errorf("unexpected content type %T", content)
}

// Next gets the immediately following sibling of each element in the
// Selection.
// If present selector gets all following siblings of each element up to but not
// including the element matched by the selector.
func Next(_ *plugin.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.NextUntil(args[0]), nil
		}
		return node.Next(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
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
func Slice(_ *plugin.Context, content any, args ...string) (any, error) {
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

	return nil, fmt.Errorf("unexpected type %T", content)
}

// Child gets the child elements of each element in the Selection.
// If present the selector will return filtered by the specified selector.
func Child(_ *plugin.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ChildrenFiltered(args[0]), nil
		}
		return node.Children(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

// Parent gets the parent of each element in the Selection.
// if present the selector will return filtered by a selector.
func Parent(_ *plugin.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ParentFiltered(args[0]), nil
		}
		return node.Parent(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

// Parents gets the ancestors of each element in the current Selection.
// if present the selector will return filtered by a selector.
func Parents(_ *plugin.Context, content any, args ...string) (any, error) {
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

	return nil, fmt.Errorf("unexpected type %T", content)
}

func Prefix(_ *plugin.Context, content any, args ...string) (ret any, err error) {
	if len(args) == 0 {
		return "", nil
	}
	if s, ok := content.(string); ok {
		return args[0] + s, nil
	} else if node, ok := content.(*goquery.Selection); ok {
		return args[0] + node.Text(), nil
	}
	return
}

func Suffix(_ *plugin.Context, content any, args ...string) (ret any, err error) {
	if len(args) == 0 {
		return "", nil
	}
	if s, ok := content.(string); ok {
		return s + args[0], nil
	} else if node, ok := content.(*goquery.Selection); ok {
		return node.Text() + args[0], nil
	}
	return
}
