package js

import (
	"flag"
	"os"
	"testing"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
)

var (
	jsParser Parser
	ctx      *plugin.Context
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = plugin.NewContext(plugin.Options{
		URL: "http://localhost/home",
	})
	code := m.Run()
	os.Exit(code)
}

func TestParser(t *testing.T) {
	if _, ok := parser.GetParser(key); !ok {
		t.Fatal("schema not registered")
	}
}

func TestGetString(t *testing.T) {
	{
		str, err := jsParser.GetString(ctx, "a", `(async () => content + 1)()`)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "a1", str)
	}

	{
		str, err := jsParser.GetString(ctx, "", `(async () => ({"test":"1"}))()`)
		if err != nil {
			t.Fatal(err)
		}

		assert.JSONEq(t, `{"test":"1"}`, str)
	}
}

func TestGetStrings(t *testing.T) {
	{
		str, err := jsParser.GetStrings(ctx, `["a1"]`,
			`new Promise((r, j) => {
					let s = JSON.parse(content);
					s.push('a2');
					r(s)
				});`)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, []string{"a1", "a2"}, str)
	}

	{
		str, err := jsParser.GetStrings(ctx, "", `[{"foo":"1"}, {"bar":"1"}, 19]`)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, []string{`{"foo":"1"}`, `{"bar":"1"}`, "19"}, str)
	}
}

func TestGetElement(t *testing.T) {
	ele, err := jsParser.GetElement(ctx, ``, `cat.setVar('size', 1 + 2);cat.getVar('size');`)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "3", ele)
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	ele, err := jsParser.GetElements(ctx, ``, `[1, 2]`)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"1", "2"}, ele)
}
