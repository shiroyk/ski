package json

import (
	"flag"
	"os"
	"testing"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/stretchr/testify/assert"
)

var (
	json    Parser
	ctx     *parser.Context
	content = `
{
    "store": {
        "book": [
            {
                "category": "reference",
                "author": "Nigel Rees",
                "title": "Sayings of the Century",
                "price": 8.95
            },
            {
                "category": "fiction",
                "author": "Evelyn Waugh",
                "title": "Sword of Honour",
                "price": 12.99
            },
            {
                "category": "fiction",
                "author": "Herman Melville",
                "title": "Moby Dick",
                "isbn": "0-553-21311-3",
                "price": 8.99
            },
            {
                "category": "fiction",
                "author": "J. R. R. Tolkien",
                "title": "The Lord of the Rings",
                "isbn": "0-395-19395-8",
                "price": 22.99
            }
        ],
        "bicycle": {
            "color": "red",
            "price": 19.95
        }
    },
    "expensive": 10
}`
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = parser.NewContext(parser.Options{})
	code := m.Run()
	os.Exit(code)
}

func assertString(t *testing.T, arg string, expected string) {
	str, err := json.GetString(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, str)
}

func assertStrings(t *testing.T, arg string, expected []string) {
	str, err := json.GetStrings(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, str)
}

func TestParser(t *testing.T) {
	if _, ok := parser.GetParser(key); !ok {
		t.Fatal("schema not registered")
	}

	contents := []any{114514, `}{`}
	for _, ct := range contents {
		if _, err := json.GetString(ctx, ct, ``); err == nil {
			t.Fatal("Unexpected type")
		}
	}

	if _, err := json.GetString(ctx, &contents[len(contents)-1], ""); err == nil {
		t.Fatal("Unexpected type")
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()
	assertString(t, `$.store.book[*].author`, `Nigel ReesEvelyn WaughHerman MelvilleJ. R. R. Tolkien`)
}

func TestGetStrings(t *testing.T) {
	t.Parallel()
	assertStrings(t, `$...book[0].price`, []string{"8.95"})

	assertStrings(t, `$...book[-1].price`, []string{"22.99"})
}

func TestGetElement(t *testing.T) {
	t.Parallel()
	if _, err := json.GetElement(ctx, content, `$$$`); err == nil {
		t.Fatal("Unexpected path")
	}

	assertString(t, `$.store.book[-1].price`, "22.99")

	str1, err := json.GetElement(ctx, content, `$.store.book[?(@.price > 20)]`)
	if err != nil {
		t.Fatal(err)
	}

	str2, err := json.GetElement(ctx, str1, `$.title`)
	if err != nil {
		t.Fatal(err)
	}
	if str2 != `The Lord of the Rings` {
		t.Fatalf("Unexpected string %s", str2)
	}
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	assertStrings(t, `$.store.book[?(@.price < 10)].isbn`, []string{`0-553-21311-3`})

	str1, err := json.GetElements(ctx, content, `$.store.book[3]`)
	if err != nil {
		t.Fatal(err)
	}

	str2, err := json.GetElement(ctx, str1[0], `$.category`)
	if err != nil {
		t.Fatal(err)
	}
	if str2 != `fiction` {
		t.Fatalf("Unexpected string %s", str2)
	}
}
