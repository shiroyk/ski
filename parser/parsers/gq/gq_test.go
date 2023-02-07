package gq

import (
	"flag"
	"os"
	"reflect"
	"testing"

	c "github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/utils"
)

var (
	gq      Parser
	ctx     *c.Context
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

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = c.NewContext(c.Options{
		Url:    "https://localhost",
		Config: c.Config{Separator: ", "},
	})
	code := m.Run()
	os.Exit(code)
}

func assertGetString(t *testing.T, arg string, assert func(string) bool) {
	str, err := gq.GetString(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	if !assert(str) {
		t.Fatalf("Unexpected string %s", str)
	}
}

func assertGetStrings(t *testing.T, arg string, assert func([]string) bool) {
	str, err := gq.GetStrings(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	if !assert(str) {
		t.Fatalf("Unexpected strings %s", str)
	}
}

func assertGetElement(t *testing.T, arg string, assert func(string2 string) bool) {
	ele, err := gq.GetElement(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	if !assert(ele) {
		t.Fatalf("Unexpected object %s", ele)
	}
}

func assertGetElements(t *testing.T, arg string, assert func([]string) bool) {
	objs, err := gq.GetElements(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	if !assert(objs) {
		t.Fatalf("Unexpected objects %s", objs)
	}
}

func TestParser(t *testing.T) {
	_, ok := c.GetParser(key)
	if !ok {
		t.Fatal("parser not registered")
	}

	if _, err := gq.GetString(ctx, 0, ``); err == nil {
		t.Fatal("expected empty error")
	}

	if _, err := gq.GetString(ctx, nil, ``); err != nil {
		t.Fatal(err)
	}

	if _, err := gq.GetString(ctx, []string{"<br>"}, ``); err != nil {
		t.Fatal(err)
	}

	if _, err := gq.GetString(ctx, utils.ToPtr(`<a href="https://go.dev" title="Golang page">Golang</a>`), ``); err != nil {
		t.Fatal(err)
	}

	sel, _ := gq.GetElement(ctx, content, `#main .row`)
	if _, err := gq.GetString(ctx, sel, ``); err != nil {
		t.Fatal(err)
	}
}

func TestGetString(t *testing.T) {
	assertGetString(t, `#main .row -> text`, func(s string) bool {
		return s == "1, 2, 3, 4, 5, 6"
	})

	assertGetString(t, `.body ul a -> parent(li) -> attr(id) -> join(-)`, func(s string) bool {
		return s == "a1-a2-a3-a4"
	})

	assertGetString(t, `script -> slice(0)`, func(s string) bool {
		return len(s) > 0
	})
}

func TestGetStrings(t *testing.T) {
	assertGetStrings(t, `.body ul li -> child(a) -> attr(title)`, func(str []string) bool {
		return reflect.DeepEqual(str, []string{"Google page", "Github page", "Golang page", "Home page"})
	})

	assertGetStrings(t, `.body ul a`, func(str []string) bool {
		return reflect.DeepEqual(str, []string{"Google", "Github", "Golang", "Home"})
	})
}

func TestGetElement(t *testing.T) {
	assertGetElement(t, `.body ul a -> parents(li)`, func(ele string) bool {
		return ele == `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>`
	})

	assertGetElement(t, `.body ul a -> slice(1) -> text`, func(ele string) bool {
		return ele == `Github`
	})
}

func TestGetElements(t *testing.T) {
	assertGetElements(t, `#foot div -> slice(0, 3)`, func(ele []string) bool {
		return reflect.DeepEqual(ele, []string{
			`<div id="nf1" class="one even row">f1</div>`,
			`<div id="nf2" class="two odd row">f2</div>`,
			`<div id="nf3" class="three even row">f3</div>`,
		})
	})

	assertGetElements(t, `#foot div -> slice(0, 3) -> text`, func(ele []string) bool {
		return reflect.DeepEqual(ele, []string{"f1", "f2", "f3"})
	})
}
