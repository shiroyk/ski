package jq

import (
	"context"
	"testing"

	"github.com/shiroyk/ski"
	"github.com/stretchr/testify/assert"
)

var (
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

func assertValue(t *testing.T, arg string, expected any) {
	exec, err := new_expr()(ski.String(arg))
	if assert.NoError(t, err) {
		v, err := exec.Exec(context.Background(), content)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, v)
		}
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()
	assertValue(t, `$.store.book[-1].price`, 22.99)
	assertValue(t, `$.store.book[*].author`, []any{"Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"})
	assertValue(t, `$.store.book[?(@.price < 10)].isbn`, []any{`0-553-21311-3`})
}
