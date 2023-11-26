package js

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
)

type testParser struct{}

func (t *testParser) GetString(_ *plugin.Context, content any, arg string) (string, error) {
	if str, ok := content.(string); ok {
		return str + arg, nil
	}
	return "", fmt.Errorf("type not supported")
}

func (t *testParser) GetStrings(_ *plugin.Context, content any, arg string) ([]string, error) {
	if str, ok := content.([]string); ok {
		return append(str, arg), nil
	}
	return nil, fmt.Errorf("type not supported")
}

func (t *testParser) GetElement(ctx *plugin.Context, content any, arg string) (string, error) {
	return t.GetString(ctx, content, arg)
}

func (t *testParser) GetElements(ctx *plugin.Context, content any, arg string) ([]string, error) {
	return t.GetStrings(ctx, content, arg)
}

func TestCtxWrapper(t *testing.T) {
	t.Parallel()
	parser.Register("test", new(testParser))
	ctx := plugin.NewContext(plugin.ContextOptions{
		URL:    "http://localhost/home",
		Logger: slog.Default(),
	})
	vm := NewTestVM(t)

	testCase := []string{
		`ctx.log('start test');`,
		`assert.equal(ctx.baseURL, "http://localhost");`,
		`assert.equal(ctx.url,"http://localhost/home");`,
		`ctx.set('v1', 114514);`,
		`assert.equal(ctx.get('v1'), 114514);`,
		`ctx.clearVar();
		 assert.equal(ctx.get('v1'), null);`,
		`assert.equal(ctx.getString('test', '1', 'foo'), 'foo1');`,
		`assert.equal(ctx.getStrings('test', '2', ['foo'])[1], '2');`,
		`assert.equal(ctx.getElement('test', '3', 'foo'), 'foo3');`,
		`assert.equal(ctx.getElements('test', '4', ['foo'])[1], '4');`,
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
