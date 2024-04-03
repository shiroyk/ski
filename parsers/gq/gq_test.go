package gq

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	gq      = parser{funcs: builtins()}
	ctx     = context.Background()
	content = `<!DOCTYPE html>
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
			<li id="a4"><a href="/home" title="Home page">Home</a></li>
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
	<script type="text/javascript">
		(function() {
		  const ga = document.createElement("script"); ga.type = "text/javascript"; ga.async = true;
		  ga.src = ("https:" === document.location.protocol ? "https://ssl" : "http://www") + ".google-analytics.com/ga.js";
		  const s = document.getElementsByTagName("script")[0]; s.parentNode.insertBefore(ga, s);
		})();
	</script>
  </body>
</html>
`
)

func assertError(t *testing.T, arg string, contains string) {
	executor, err := gq.Value(arg)
	if assert.NoError(t, err) {
		_, err = executor.Exec(ctx, content)
		assert.ErrorContains(t, err, contains)
	}
}

func assertValue(t *testing.T, arg string, expected any) {
	executor, err := gq.Value(arg)
	if assert.NoError(t, err) {
		v, err := executor.Exec(ctx, content)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, v)
		}
	}
}

func assertElement(t *testing.T, arg string, expected string) {
	executor, err := gq.Element(arg)
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
	executor, err := gq.Elements(arg)
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
	assertValue(t, `#main .row -> text`, []string{"1", "2", "3", "4", "5", "6"})

	assertValue(t, `.body ul a -> parent(li) -> attr(id)`, []string{"a1", "a2", "a3", "a4"})

	assertValue(t, `script -> slice(0) -> attr(type)`, "text/javascript")
}

func TestElement(t *testing.T) {
	t.Parallel()
	assertElement(t, `.body ul a -> parents(li)`, `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>`)

	assertElement(t, `.body ul a -> slice(1) -> text`, `Github`)
}

func TestElements(t *testing.T) {
	t.Parallel()
	assertElements(t, `#foot div -> slice(0, 3)`, []string{
		`<div id="nf1" class="one even row">f1</div>`,
		`<div id="nf2" class="two odd row">f2</div>`,
		`<div id="nf3" class="three even row">f3</div>`,
	})

	assertElements(t, `#foot div -> slice(0, 3) -> text`, []string{"f1", "f2", "f3"})
}

func TestNodeSelect(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		exec, err := gq.Element(`script -> slice(0)`)
		if !assert.NoError(t, err) {
			return
		}
		v, err := exec.Exec(ctx, content)
		if !assert.NoError(t, err) {
			return
		}
		{
			exec, err = gq.Value(`-> attr(type)`)
			if !assert.NoError(t, err) {
				return
			}
			v1, err := exec.Exec(ctx, v)
			if assert.NoError(t, err) {
				assert.Equal(t, "text/javascript", v1)
			}
		}
		{
			exec, err = gq.Value(`script -> attr(type)`)
			if !assert.NoError(t, err) {
				return
			}
			v2, err := exec.Exec(ctx, v)
			if assert.NoError(t, err) {
				assert.Equal(t, "text/javascript", v2)
			}
		}
	})

	t.Run("multiple", func(t *testing.T) {
		exec, err := gq.Elements(`#foot div -> slice(0, 3)`)
		if !assert.NoError(t, err) {
			return
		}
		v, err := exec.Exec(ctx, content)
		if !assert.NoError(t, err) {
			return
		}
		{
			exec, err = gq.Value(`-> text`)
			if !assert.NoError(t, err) {
				return
			}
			v1, err := exec.Exec(ctx, v)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"f1", "f2", "f3"}, v1)
			}
		}
		{
			exec, err = gq.Value(`div -> text`)
			if !assert.NoError(t, err) {
				return
			}
			v2, err := exec.Exec(ctx, v)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"f1", "f2", "f3"}, v2)
			}
		}
	})
}

func TestExternalFunc(t *testing.T) {
	{
		fun := func(logger *slog.Logger) Func {
			return func(_ context.Context, content any, args ...string) (any, error) {
				logger.Info(fmt.Sprintf("result type was %T", content))
				return content, nil
			}
		}
		data := new(bytes.Buffer)
		p := NewParser(FuncMap{"logger": fun(slog.New(slog.NewTextHandler(data, nil)))})
		executor, err := p.Value(".body ul a -> logger -> text")
		if assert.NoError(t, err) {
			v, err := executor.Exec(ctx, content)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"Google", "Github", "Golang", "Home"}, v)
			}
		}
		assert.Contains(t, data.String(), `result type was *goquery.Selection`)
	}

	{
		fun := func(_ context.Context, content any, args ...string) (any, error) {
			return nil, nil
		}
		p := NewParser(FuncMap{"nil": fun})
		executor, err := p.Value(".body ul a -> nil -> text")
		if assert.NoError(t, err) {
			v, err := executor.Exec(ctx, content)
			if assert.NoError(t, err) {
				assert.Equal(t, nil, v)
			}
		}
	}
}
