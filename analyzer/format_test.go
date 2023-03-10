package analyzer

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/shiroyk/cloudcat/schema"
)

func TestFormat(t *testing.T) {
	t.Parallel()
	formatter := new(defaultFormatHandler)
	testCases := []struct {
		data any
		typ  schema.Type
		want any
	}{
		{"1", schema.StringType, "1"},
		{"2.1", schema.NumberType, 2.1},
		{"3", schema.IntegerType, 3},
		{"1", schema.BooleanType, true},
		{`{"k":"v"}`, schema.ObjectType, map[string]any{"k": "v"}},
		{`[1,2]`, schema.ArrayType, []any{1.0, 2.0}},
		{[]string{"1", "2"}, schema.IntegerType, []any{1, 2}},
		{map[string]any{"k": "1"}, schema.IntegerType, map[string]any{"k": 1}},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got, err := formatter.Format(testCase.data, testCase.typ)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
