package gq

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/spf13/cast"
)

type (
	BuildInFunc struct{}
	Func        func(ctx *parser.Context, content any, args ...string) (any, error)
)

var (
	buildInFunc  BuildInFunc
	buildInFuncs = map[string]Func{
		"get":     buildInFunc.Get,
		"set":     buildInFunc.Set,
		"attr":    buildInFunc.Attr,
		"href":    buildInFunc.Href,
		"html":    buildInFunc.Html,
		"join":    buildInFunc.Join,
		"prev":    buildInFunc.Prev,
		"text":    buildInFunc.Text,
		"next":    buildInFunc.Next,
		"slice":   buildInFunc.Slice,
		"child":   buildInFunc.Child,
		"parent":  buildInFunc.Parent,
		"parents": buildInFunc.Parents,
	}
)

func mapToString(content any, f func(*goquery.Selection) (string, error)) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		list := make([]string, node.Length())
		node.EachWithBreak(func(i int, sel *goquery.Selection) bool {
			result, err := f(sel)
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

func (BuildInFunc) Get(ctx *parser.Context, _ any, args ...string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("get function must has one argment")
	}

	key, err := cast.ToStringE(args[0])
	if err != nil {
		return nil, err
	}

	return ctx.Value(key), nil
}

func (BuildInFunc) Set(ctx *parser.Context, content any, args ...string) (any, error) {
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

func (BuildInFunc) Text(_ *parser.Context, content any, _ ...string) (any, error) {
	return mapToString(content, func(node *goquery.Selection) (string, error) {
		return strings.TrimSpace(node.Text()), nil
	})
}

func (f BuildInFunc) Join(ctx *parser.Context, content any, args ...string) (any, error) {
	if str, ok := content.(string); ok {
		return str, nil
	}

	if node, ok := content.(*goquery.Selection); ok {
		text, err := f.Text(ctx, node)
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

	sep := ctx.Config().Separator
	if len(args) > 0 {
		sep = args[0]
	}

	return strings.Join(list, sep), nil
}

func (BuildInFunc) Attr(_ *parser.Context, content any, args ...string) (any, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("attr(name) must has name")
	}

	name := args[0]
	defaultValue := ""
	if len(args) > 1 {
		defaultValue = args[1]
	}

	return mapToString(content, func(node *goquery.Selection) (string, error) {
		return node.AttrOr(name, defaultValue), nil
	})
}

func (BuildInFunc) Href(ctx *parser.Context, content any, _ ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		hrefUrl, err := url.Parse(node.AttrOr("href", ""))
		if err != nil {
			return nil, err
		}

		baseUrl, err := url.Parse(ctx.BaseUrl())
		if err != nil {
			return nil, err
		}
		return baseUrl.ResolveReference(hrefUrl), nil
	}

	return nil, fmt.Errorf("unexpected content type %T", content)
}

func (BuildInFunc) Html(_ *parser.Context, content any, args ...string) (any, error) {
	var err error
	outer := false

	if len(args) > 0 {
		outer, err = cast.ToBoolE(args[0])
		if err != nil {
			return nil, fmt.Errorf("html(outer) `outer` must bool type value: true/false")
		}
	}

	return mapToString(content, func(node *goquery.Selection) (string, error) {
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

func (BuildInFunc) Prev(_ *parser.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.PrevUntil(args[0]), nil
		}
		return node.Prev(), nil
	}

	return nil, fmt.Errorf("unexpected content type %T", content)
}

func (BuildInFunc) Next(_ *parser.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.NextUntil(args[0]), nil
		}
		return node.Next(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

func (BuildInFunc) Slice(_ *parser.Context, content any, args ...string) (any, error) {
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
		} else {
			return node.Eq(start), nil
		}
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

func (BuildInFunc) Child(_ *parser.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ChildrenFiltered(args[0]), nil
		}
		return node.Children(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

func (BuildInFunc) Parent(_ *parser.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
		if len(args) > 0 {
			return node.ParentFiltered(args[0]), nil
		}
		return node.Parent(), nil
	}

	return nil, fmt.Errorf("unexpected type %T", content)
}

func (BuildInFunc) Parents(_ *parser.Context, content any, args ...string) (any, error) {
	if node, ok := content.(*goquery.Selection); ok {
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
