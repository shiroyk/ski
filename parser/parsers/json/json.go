// Package json the json parser
package json

import (
	"fmt"
	"strings"

	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	"github.com/shiroyk/cloudcat/parser"
)

// Parser the json parser
type Parser struct{}

const key string = "json"

func init() {
	parser.Register(key, new(Parser))
}

// GetString gets the string of the content with the given arguments.
//
// content := `{"keys": [{"key":"foo"},{"key":"bar"}]}`
// GetString(ctx, content, "$.key[*].key") returns "foo\nbar"
func (p Parser) GetString(_ *parser.Context, content any, arg string) (string, error) {
	obj, err := getDoc(content, arg)
	if err != nil {
		return "", err
	}

	str := make([]string, len(obj))
	var ok bool

	for i, o := range obj {
		if str[i], ok = o.(string); !ok {
			str[i] = oj.JSON(o)
		}
	}

	return strings.Join(str, "\n"), nil
}

// GetStrings gets the strings of the content with the given arguments.
//
// content := `{"keys": [{"key":"foo"},{"key":"bar"}]}`
// GetStrings(ctx, content, "$.key[*].key") returns []string{"foo", "bar"}
func (p Parser) GetStrings(_ *parser.Context, content any, arg string) ([]string, error) {
	obj, err := getDoc(content, arg)
	if err != nil {
		return nil, err
	}

	str := make([]string, len(obj))
	var ok bool

	for i, o := range obj {
		if str[i], ok = o.(string); !ok {
			str[i] = oj.JSON(o)
		}
	}

	return str, nil
}

// GetElement gets the element of the content with the given arguments.
// sames as the GetString.
func (p Parser) GetElement(ctx *parser.Context, content any, arg string) (string, error) {
	return p.GetString(ctx, content, arg)
}

// GetElements gets the elements of the content with the given arguments.
// sames as the GetStrings.
func (p Parser) GetElements(ctx *parser.Context, content any, arg string) ([]string, error) {
	return p.GetStrings(ctx, content, arg)
}

func getDoc(content any, arg string) ([]any, error) {
	var err error
	var doc any
	switch data := content.(type) {
	default:
		return nil, fmt.Errorf("unexpected content type %T", content)
	case nil:
		return []any{}, nil
	case []string:
		if len(data) == 0 {
			return nil, fmt.Errorf("unexpected content %s", content)
		}
		if doc, err = oj.ParseString(data[0]); err != nil {
			return nil, err
		}
	case string:
		if doc, err = oj.ParseString(data); err != nil {
			return nil, err
		}
	}

	x, err := jp.ParseString(arg)
	if err != nil {
		return nil, err
	}

	return x.Get(doc), nil
}
