package schema

import (
	"strconv"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type testParser struct{}

func (t *testParser) GetString(_ *parser.Context, content any, arg string) (string, error) {
	if str, ok := content.(string); ok {
		if str == arg {
			return str, nil
		}
	}
	return "", nil
}

func (t *testParser) GetStrings(_ *parser.Context, content any, arg string) ([]string, error) {
	if str, ok := content.(string); ok {
		if str == arg {
			return []string{str}, nil
		}
	}
	return nil, nil
}

func (t *testParser) GetElement(ctx *parser.Context, content any, arg string) (string, error) {
	return t.GetString(ctx, content, arg)
}

func (t *testParser) GetElements(ctx *parser.Context, content any, arg string) ([]string, error) {
	return t.GetStrings(ctx, content, arg)
}

func TestActions(t *testing.T) {
	t.Parallel()

	if _, ok := parser.GetParser("act"); !ok {
		parser.Register("act", new(testParser))
	}
	ctx := parser.NewContext(parser.Options{Timeout: time.Second})

	testCases := []struct {
		acts, content string
		want          any
		str           bool
	}{
		{`
- act: 1
- and
- act: 1
`, `1`, `11`, true,
		},
		{`
- act: 1
- and
- act: 1
`, `1`, []string{`1`, `1`}, false,
		},
		{`
- act: 2
- or
- act: 1
`, `1`, `1`, true,
		},
		{`
- act: 2
- or
- act: 1
`, `1`, []string{`1`}, false,
		},
		{`
- act: 1
- and
- act: 2
- or
- act: 1
`, `1`, `11`, true,
		},
		{`
- act: 1
- and
- act: 2
- or
- act: 1
`, `1`, []string{`1`, `1`}, false,
		},
		{`
- - act: 1
    act: 1
- and
- - act: 2
- or
- - act: 1
`, `1`, `11`, true,
		},
		{`
- - act: 1
    act: 1
- and
- - act: 2
- or
- - act: 1
`, `1`, []string{`1`}, false,
		},
	}
	if _, ok := parser.GetParser("act"); !ok {
		parser.Register("act", new(testParser))
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var act Actions
			err := yaml.Unmarshal([]byte(testCase.acts), &act)
			if err != nil {
				t.Error(err)
			}
			var result any
			if testCase.str {
				result, err = act.GetString(ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			} else {
				result, err = act.GetStrings(ctx, testCase.content)
				if err != nil {
					t.Error(err)
				}
			}
			assert.Equal(t, testCase.want, result)
		})
	}

}
