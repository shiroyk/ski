package js

import (
	"context"
	"flag"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/fetcher"
	p "github.com/shiroyk/cloudcat/parser"
)

var (
	js  Parser
	ctx *p.Context
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = p.NewContext(&p.Options{
		Url:       "http://localhost/home",
		Cookie:    memory.NewCookie(),
		Cache:     memory.NewCache(),
		Shortener: memory.NewShortener(),
		Fetcher:   fetcher.NewFetcher(&fetcher.Options{}),
	})
	code := m.Run()
	os.Exit(code)
}

func TestParser(t *testing.T) {
	p.GetDesc(key)
	if r := recover(); r != nil {
		t.Fatal(r)
	}
}

func TestGetString(t *testing.T) {
	str, err := js.GetString(ctx, "a", `(async () => go.content + 1)()`)
	if err != nil {
		t.Fatal(err)
	}
	if str != "a1" {
		t.Errorf("unexpected result %s", str)
	}
}

func TestGetStrings(t *testing.T) {
	str, err := js.GetStrings(ctx, `["a1"]`,
		`new Promise((r, j) => {
				let s = JSON.parse(go.content);
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
	ele, err := js.GetElement(ctx, ``,
		`go.setVar('size', 1 + 2);go.getVar('size')`)
	if err != nil {
		t.Fatal(err)
	}
	if ele != "3" {
		t.Fatalf("unexpected result %s", ele)
	}
}

func TestGetElements(t *testing.T) {
	ele, err := js.GetElements(ctx, ``,
		`[1, 2]`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ele, []string{"1", "2"}) {
		t.Fatalf("unexpected result %s", ele)
	}
}

func TestTimeout(t *testing.T) {
	_, err := js.GetString(p.NewContext(&p.Options{
		Timeout: time.Second * 1,
	}), ``, `while(true){}`)
	if err != nil {
		if face, ok := err.(*goja.InterruptedError); ok {
			if e := face.Unwrap(); e != context.DeadlineExceeded {
				t.Fatalf("unexpected error: %s", e)
			}
		} else {
			t.Fatalf("unexpected error: %s", err)
		}
	}
}
