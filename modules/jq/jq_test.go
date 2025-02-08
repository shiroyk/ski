package jq

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestJq(t *testing.T) {
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		v, _ := Jq{}.Instantiate(rt)
		_ = rt.Set("jq", v)
	}))
	ctx := context.Background()

	t.Run("basic queries", func(t *testing.T) {
		cases := []struct {
			expr     string
			expected any
		}{
			{`$.store.book[0].title`, []any{"Sayings of the Century"}},
			{`$.store.book[*].author`, []any{
				"Nigel Rees",
				"Evelyn Waugh",
				"Herman Melville",
				"J. R. R. Tolkien",
			}},
			{`$.store.book[?(@.price < 10)].title`, []any{
				"Sayings of the Century",
				"Moby Dick",
			}},
			{`$.store.bicycle.color`, []any{"red"}},
			{`$.expensive`, []any{int64(10)}},
		}

		for _, tc := range cases {
			t.Run(tc.expr, func(t *testing.T) {
				result, err := vm.RunString(ctx, `jq('`+tc.expr+`').get(`+content+`);`)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			})
		}
	})

	t.Run("first", func(t *testing.T) {
		result, err := vm.RunString(ctx, `
			jq('$.store.book[*].author').first(`+content+`);
		`)
		require.NoError(t, err)
		assert.Equal(t, "Nigel Rees", result.Export())
	})

	t.Run("set", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let data = JSON.parse(`+"`"+content+"`"+`);
			let expr = jq('$.store.bicycle.color');
			expr.set(data, "blue");
			return expr.get(data);
		}`)
		require.NoError(t, err)
		assert.Equal(t, []any{"blue"}, result.Export())
	})

	t.Run("setOne", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let data = JSON.parse(`+"`"+content+"`"+`);
			let expr = jq('$.store.book[*].price');
			expr.setOne(data, 9.99);
			return expr.first(data);
		}`)
		require.NoError(t, err)
		assert.Equal(t, 9.99, result.Export())
	})

	t.Run("has", func(t *testing.T) {
		cases := []struct {
			expr     string
			expected bool
		}{
			{`$.store.book[0].isbn`, false},
			{`$.store.book[2].isbn`, true},
			{`$.store.bicycle.brand`, false},
			{`$.store.bicycle.color`, true},
		}

		for _, tc := range cases {
			t.Run(tc.expr, func(t *testing.T) {
				result, err := vm.RunString(ctx, `jq('`+tc.expr+`').has(`+content+`);`)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			})
		}
	})

	t.Run("remove", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let data = JSON.parse(`+"`"+content+"`"+`);
			let expr = jq('$.store.book[?(@.category == "fiction")]');
			expr.remove(data);
			let remaining = jq('$.store.book[*].category').get(data);
			return remaining;
		}`)
		require.NoError(t, err)
		assert.Equal(t, []any{"reference"}, result.Export())
	})

	t.Run("removeOne", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			let data = JSON.parse(`+"`"+content+"`"+`);
			let expr = jq('$.store.book[?(@.category == "fiction")]');
			expr.removeOne(data);
			let remaining = jq('$.store.book[*].author').get(data);
			return remaining.length;
		}`)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Export())
	})

	t.Run("complex queries", func(t *testing.T) {
		cases := []struct {
			expr     string
			expected any
		}{
			{`$.store.book[?(@.price > 10)].title`, []any{
				"Sword of Honour",
				"The Lord of the Rings",
			}},
			{`$.store.book[?(@.category == "fiction" && @.price < 10)].title`, []any{
				"Moby Dick",
			}},
			{`$.store.book[?(@.price > $.expensive)].title`, []any{
				"Sword of Honour",
				"The Lord of the Rings",
			}},
		}

		for _, tc := range cases {
			t.Run(tc.expr, func(t *testing.T) {
				result, err := vm.RunString(ctx, `
					jq('`+tc.expr+`').get(`+content+`);
				`)
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
				`jq('$[invalid')`,
			},
			{
				"invalid json",
				`jq('$.store').get('{invalid json}')`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := vm.RunString(ctx, tc.code)
				assert.Error(t, err)
			})
		}
	})
}
