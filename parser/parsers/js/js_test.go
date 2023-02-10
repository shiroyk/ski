package js

import (
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/shiroyk/cloudcat/schema/parsers"
)

var (
	jsParser Parser
	ctx      *parsers.Context
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = parsers.NewContext(parsers.Options{
		URL: "http://localhost/home",
	})
	code := m.Run()
	os.Exit(code)
}

func TestParser(t *testing.T) {
	_, ok := parsers.GetParser(key)
	if !ok {
		t.Fatal("schema not registered")
	}
}

func TestGetString(t *testing.T) {
	str, err := jsParser.GetString(ctx, "a", `(async () => content + 1)()`)
	if err != nil {
		t.Fatal(err)
	}
	if str != "a1" {
		t.Errorf("unexpected result %s", str)
	}
}

func TestGetStrings(t *testing.T) {
	str, err := jsParser.GetStrings(ctx, `["a1"]`,
		`new Promise((r, j) => {
				let s = JSON.parse(content);
		   		s.push('a2');
				r(s)
		   });`)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(str, [2]string{"a1", "a2"}) {
		t.Errorf("unexpected result %s", str)
	}
}

func TestGetElement(t *testing.T) {
	ele, err := jsParser.GetElement(ctx, ``,
		`cat.setVar('size', 1 + 2);cat.getVar('size');`)
	if err != nil {
		t.Fatal(err)
	}
	if ele != "3" {
		t.Fatalf("unexpected result %s", ele)
	}
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	ele, err := jsParser.GetElements(ctx, ``,
		`[1, 2]`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ele, []string{"1", "2"}) {
		t.Fatalf("unexpected result %s", ele)
	}
}
