package xpath

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	p       Parser
	ctx     = context.Background()
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

func assertError(t *testing.T, arg string, contains string) {
	_, err := p.Value(arg)
	assert.ErrorContains(t, err, contains)
}

func assertValue(t *testing.T, arg string, expected any) {
	executor, err := p.Value(arg)
	if assert.NoError(t, err) {
		v, err := executor.Exec(ctx, content)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, v)
		}
	}
}

func assertElement(t *testing.T, arg string, expected string) {
	executor, err := p.Element(arg)
	if assert.NoError(t, err) {
		v, err := executor.Exec(ctx, content)
		if assert.NoError(t, err) {
			switch c := v.(type) {
			case *html.Node:
				b := new(bytes.Buffer)
				if assert.NoError(t, html.Render(b, c)) {
					assert.Equal(t, expected, b.String())
				}
			default:
				assert.Equal(t, expected, v)
			}
		}
	}
}

func assertElements(t *testing.T, arg string, expected []string) {
	executor, err := p.Elements(arg)
	if assert.NoError(t, err) {
		v, err := executor.Exec(ctx, content)
		if assert.NoError(t, err) {
			switch c := v.(type) {
			case []any:
				ele := make([]string, len(c))
				for i, v := range c {
					var b bytes.Buffer
					if assert.NoError(t, html.Render(&b, v.(*html.Node))) {
						ele[i] = b.String()
					}
				}
				assert.Equal(t, expected, ele)
			default:
				assert.Equal(t, expected, v)
			}
		}
	}
}

func TestValue(t *testing.T) {
	t.Parallel()
	assertError(t, `///`, "expression must evaluate to a node-set")

	assertValue(t, `//div[@id="main"]/div[contains(@class, "row")]/text()`, []string{"1", "2", "3", "4", "5", "6"})

	assertValue(t, `//div[@class="body"]/ul/li/@id`, []string{"a1", "a2", "a3"})

	assertValue(t, `//script[1]`, `(function() {})();`)
}

func TestElement(t *testing.T) {
	t.Parallel()

	assertElement(t, `//div[@class="body"]/ul//a/..`, `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>`)
}

func TestElements(t *testing.T) {
	t.Parallel()

	assertElements(t, `//div[@id="foot"]/div/@class`, []string{
		"<class>one even row</class>", "<class>two odd row</class>",
		"<class>three even row</class>", "<class>four odd row</class>",
		"<class>five even row odder</class>", "<class>six odd row</class>",
	})
}
