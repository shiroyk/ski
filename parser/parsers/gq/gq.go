package gq

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/utils"
	"github.com/spf13/cast"
)

type Parser struct{}

const key string = "gq"

func init() {
	parser.Register(key, new(Parser))
}

func (p Parser) GetString(ctx *parser.Context, content any, arg string) (ret string, err error) {
	nodes, err := getSelection(content)
	if err != nil {
		return
	}

	rule, funcs, err := parseRuleFunctions(arg)
	if err != nil {
		return
	}

	selection := nodes.Find(rule)
	var node any
	node = selection

	for _, fun := range funcs {
		if node, err = buildInFuncs[fun.name](ctx, node, fun.args...); err != nil {
			return ret, err
		}
	}

	join, err := buildInFunc.Join(ctx, node)
	if err != nil {
		return
	}

	return cast.ToStringE(join)
}

func (p Parser) GetStrings(ctx *parser.Context, content any, arg string) (ret []string, err error) {
	nodes, err := getSelection(content)
	if err != nil {
		return
	}

	rule, funcs, err := parseRuleFunctions(arg)
	if err != nil {
		return
	}

	selection := nodes.Find(rule)
	var node any
	node = selection

	for _, fun := range funcs {
		if node, err = buildInFuncs[fun.name](ctx, node, fun.args...); err != nil {
			return nil, err
		}
	}

	sel, ok := node.(*goquery.Selection)
	if ok {
		str := make([]string, sel.Length())
		var err error
		sel.EachWithBreak(func(i int, sel *goquery.Selection) bool {
			str[i] = strings.TrimSpace(sel.Text())
			return true
		})
		if err != nil {
			return ret, err
		}
		return str, nil
	}
	return cast.ToStringSliceE(node)
}

func (p Parser) GetElement(ctx *parser.Context, content any, arg string) (ret string, err error) {
	nodes, err := getSelection(content)
	if err != nil {
		return
	}

	rule, funcs, err := parseRuleFunctions(arg)
	if err != nil {
		return
	}

	var node any
	node = nodes.Find(rule)

	for _, fun := range funcs {
		if node, err = buildInFuncs[fun.name](ctx, node, fun.args...); err != nil {
			return ret, err
		}
	}

	if sel, ok := node.(*goquery.Selection); ok {
		return goquery.OuterHtml(sel)
	}

	return cast.ToStringE(node)
}

func (p Parser) GetElements(ctx *parser.Context, content any, arg string) (ret []string, err error) {
	nodes, err := getSelection(content)
	if err != nil {
		return
	}

	rule, funcs, err := parseRuleFunctions(arg)
	if err != nil {
		return
	}

	selection := nodes.Find(rule)
	var node any
	node = selection

	for _, fun := range funcs {
		if node, err = buildInFuncs[fun.name](ctx, node, fun.args...); err != nil {
			return nil, err
		}
	}

	sel, ok := node.(*goquery.Selection)
	if ok {
		objs := make([]string, sel.Length())
		sel.EachWithBreak(func(i int, sel *goquery.Selection) bool {
			if objs[i], err = goquery.OuterHtml(sel); err != nil {
				return false
			}
			return true
		})
		if err != nil {
			return
		}
		return objs, nil
	}
	return cast.ToStringSliceE(node)
}

// getSelection converts content to goquery.Selection
func getSelection(content any) (*goquery.Selection, error) {
	switch data := utils.FromPtr(content).(type) {
	default:
		return nil, fmt.Errorf("unexpected content type %T", data)
	case nil:
		return &goquery.Selection{}, nil
	case []string:
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(strings.Join(data, "\n")))
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
