package xpath

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

var (
	content = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>Tests for siblings</title>
  </head>
  <body>
    <div id="main">
      <div id="n1" class="one even row">1</div>
      <div id="n2" class="two odd row">2</div>
      <div id="n3" class="three even row">3</div>
      <div id="n4" class="four odd row">4</div>
      <div id="n5" class="five even row">5</div>
      <div id="n6" class="six odd row">6</div>
    </div>
	<div class="body">
        <ul id="url">
			<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>
			<li id="a2"><a href="https://github.com" title="Github page">Github</a></li>
			<li id="a3" class="selected"><a href="https://go.dev" title="Golang page">Golang</a></li>
		</ul>
	</div>
    <div id="foot">
      <div id="nf1" class="one even row">f1</div>
      <div id="nf2" class="two odd row">f2</div>
      <div id="nf3" class="three even row">f3</div>
      <div id="nf4" class="four odd row">f4</div>
      <div id="nf5" class="five even row odder">f5</div>
      <div id="nf6" class="six odd row">f6</div>
    </div>
	<script type="text/javascript">(function() {})();</script>
  </body>
</html>
`
)

func TestXpath(t *testing.T) {
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		v, _ := new(Xpath).Instantiate(rt)
		_ = rt.Set("xpath", v)
	}))
	ctx := context.Background()

	t.Run("basic queries", func(t *testing.T) {
		cases := []struct {
			expr     string
			expected []string
		}{
			{`//title`, []string{"Tests for siblings"}},
			{`//div[@id="main"]/div[@class="one even row"]`, []string{"1"}},
			{`//div[@id="main"]/div[contains(@class, "odd")]`, []string{"2", "4", "6"}},
			{`//div[@id="foot"]/div[contains(@class, "even")]`, []string{"f1", "f3", "f5"}},
			{`//ul[@id="url"]/li/a/@href`, []string{
				"https://google.com",
				"https://github.com",
				"https://go.dev",
			}},
		}

		for _, tc := range cases {
			t.Run(tc.expr, func(t *testing.T) {
				result, err := vm.RunString(ctx, `
					xpath('`+tc.expr+`').innerText(`+"`"+content+"`"+`);`)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			})
		}
	})

	t.Run("element queries", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let doc = `+"`"+content+"`"+`;
			let expr = xpath('//ul[@id="url"]/li[@class="selected"]/a');
			let el = expr.querySelector(doc);
			return el.attr;
		}`)
		require.NoError(t, err)
		assert.Equal(t, &[]html.Attribute{{Key: "href", Val: "https://go.dev"}, {Key: "title", Val: "Golang page"}}, result.Export())
	})

	t.Run("elements queries", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let doc = `+"`"+content+"`"+`;
			let expr = xpath('//div[@id="main"]/div[position() mod 2 = 0]');
			let elements = expr.querySelectorAll(doc);
			return elements.map(el => el.firstChild.data);
		}`)
		require.NoError(t, err)
		assert.Equal(t, []any{"2", "4", "6"}, result.Export())
	})

	t.Run("complex queries", func(t *testing.T) {
		cases := []struct {
			expr     string
			expected []string
		}{
			{`//div[contains(@class, "even") and contains(@class, "row")]`, []string{
				"1", "3", "5", "f1", "f3", "f5",
			}},
			{`//div[@id="foot"]/div[position() < 3]`, []string{"f1", "f2"}},
			{`//li[a[contains(@href, "github.com")]]`, []string{"Github"}},
			{`//div[@id="main"]/div[last()]`, []string{"6"}},
		}

		for _, tc := range cases {
			t.Run(tc.expr, func(t *testing.T) {
				result, err := vm.RunString(ctx, `
					xpath('`+tc.expr+`').innerText(`+"`"+content+"`"+`);`)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			})
		}
	})

	t.Run("error handling", func(t *testing.T) {
		cases := []struct {
			name string
			code string
		}{
			{
				"invalid expression",
				`xpath('//div[')`,
			},
			{
				"empty result",
				`xpath('//nonexistent').querySelectorAll(` + "`" + content + "`" + `)`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tc.code)
				if err == nil {
					assert.Equal(t, "null", result.String())
				} else {
					assert.Error(t, err)
				}
			})
		}
	})
}
