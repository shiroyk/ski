package js

import (
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/schema"
	"github.com/shiroyk/cloudcat/schema/parsers"
	_ "github.com/shiroyk/cloudcat/schema/parsers/json"
	"golang.org/x/exp/slog"
)

type testParser struct{}

func (t *testParser) GetString(_ *parsers.Context, content any, arg string) (string, error) {
	if str, ok := content.(string); ok {
		return str + arg, nil
	}
	return "", fmt.Errorf("type not supported")
}

func (t *testParser) GetStrings(ctx *parsers.Context, content any, arg string) ([]string, error) {
	str, err := t.GetString(ctx, content, arg)
	if err != nil {
		return nil, err
	}
	return []string{str}, nil
}

func (t *testParser) GetElement(ctx *parsers.Context, content any, arg string) (string, error) {
	return t.GetString(ctx, content, arg)
}

func (t *testParser) GetElements(ctx *parsers.Context, content any, arg string) ([]string, error) {
	return t.GetStrings(ctx, content, arg)
}

func TestCat(t *testing.T) {
	t.Parallel()
	schema.Register("test", new(testParser))
	ctx := parsers.NewContext(parsers.Options{
		URL:    "http://localhost/home",
		Logger: slog.Default().WithGroup("js"),
	})
	vm := newVM(true)

	testCase := []string{
		`cat.log('start test')`,
		`if (cat.baseURL !== "http://localhost") throw ("not equal, got" + cat.baseURL);`,
		`if (cat.redirectURL !== "http://localhost/home") throw ("not equal, got" + cat.redirectURL);`,
		`cat.setVar('v1', 114514)`,
		`if (cat.getVar('v1') !== 114514) throw ("not equal, got" + cat.getVar('v1'));`,
		`cat.clearVar()
		 if (cat.getVar('v1')) throw ("variable should be cleared");`,
		`if (cat.getString('test', 'foo', '1') !== 'foo1') throw ("unexpect result");`,
		`if (cat.getStrings('test', 'foo', '2')[0] !== 'foo2') throw ("unexpect result");`,
		`if (cat.getElement('test', 'foo', '3') !== 'foo3') throw ("unexpect result");`,
		`if (cat.getElements('test', 'foo', '4')[0] !== 'foo4') throw ("unexpect result");`,
	}
	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
