package regex

import (
	"context"
	"testing"

	"github.com/shiroyk/ski"
	"github.com/stretchr/testify/assert"
)

func TestReplace(t *testing.T) {
	t.Parallel()
	testCases := []struct{ re, str, want string }{
		{`/[0-9]/`, `114i`, "i"},
		{`/[0-9]/i/`, `114`, "iii"},
		{`/\\//`, `1/`, "1"},
		{`/[a-z]/1/`, `aaa`, "111"},
		{`/olang/olang/i`, `GoLAnG`, "Golang"},
		{`/[^ ]+\s(?<time>)/${time}/`, `08/10/99 16:00`, "16:00"},
		{`/D\.(.+)/David $1/`, `D.Bau`, "David Bau"},
		{`/a/b/{0,2}`, `aaaaa`, "bbaaa"},
		{`/a/b/{3,2}`, `aaaaa`, "aaabb"},
		{`/[a-z]/1/i{3,3}`, `aaaBBB`, "aaa111"},
		{`/a/stuff/{-1,-1}`, `a test a blah and a`, "stuff test stuff blstuffh stuffnd stuff"},
		{`/(\p{Sc}\s?)?(\d+\.?((?<=\.)\d+)?)(?(1)|\s?\p{Sc})?/$2/`, `$17.43  €2 16.33  £0.98  0.43   £43   12€  17`, "17.43  2 16.33  0.98  0.43   43   12  17"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.re, func(t *testing.T) {
			exec, err := new_replace()(ski.String(testCase.re))
			if err != nil {
				t.Fatal(err)
			}
			v, err := exec.Exec(context.Background(), testCase.str)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, testCase.want, v)
		})
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		re, str string
		want    any
	}{
		{`/\d+/`, `114a514b1919`, "114"},
		{`/\d+/1`, `114a514b1919`, "514"},
		{`/\d+/-1`, `114a514b1919`, "1919"},
		{`/\d+/-1919`, `114a514b1919`, nil},
		{`/\d+/{-114,-1919}`, `114a514b1919`, nil},
		{`/\d+/{0,1}`, `114a514b1919`, []string{"114", "514"}},
		{`/\d+/{5,6}`, `114a514b1919`, nil},
		{`/[a-z]+/{0,2}`, `114a514b1919c810`, []string{"a", "b", "c"}},
		{`/[a-z]+/i{1,2}`, `114A514B1919C810`, []string{"B", "C"}},
		{`/\\//`, `1/`, "/"},
		{`/[a-z]/`, `aaa`, "a"},
		{`/[a-z]+/`, `ABCxyz`, "xyz"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.re, func(t *testing.T) {
			exec, err := new_match()(ski.String(testCase.re))
			if err != nil {
				t.Fatal(err)
			}
			v, err := exec.Exec(context.Background(), testCase.str)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, testCase.want, v)
		})
	}
}

func TestAssert(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		re, str string
		valid   bool
	}{
		{`/[0-9]/`, `114i`, true},
		{`/[a-z]/`, `abc`, true},
		{`/[a-z]/i`, `ABC`, true},
		{`/[a-z]/`, `123`, false},
		{`/[a-z]/check failed/`, `123`, false},
		{`/[a-z]/check failed/i`, `123`, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.re, func(t *testing.T) {
			exec, err := new_assert()(ski.String(testCase.re))
			if err != nil {
				t.Fatal(err)
			}
			_, err = exec.Exec(context.Background(), testCase.str)
			if testCase.valid {
				assert.NoError(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
