package regex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	re       Parser
	testCase = []struct{ re, str, want string }{
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
)

func TestValue(t *testing.T) {
	t.Parallel()
	for _, s := range testCase {
		t.Run(s.re, func(t *testing.T) {
			executor, err := re.Value(s.re)
			if assert.NoError(t, err) {
				v, err := executor.Exec(context.Background(), s.str)
				if assert.NoError(t, err) {
					assert.Equal(t, s.want, v)
				}
			}
		})
	}
}

func TestElements(t *testing.T) {
	t.Parallel()
	for _, s := range testCase {
		t.Run(s.re, func(t *testing.T) {
			executor, err := re.Elements(s.re)
			if assert.NoError(t, err) {
				v, err := executor.Exec(context.Background(), s.str)
				if assert.NoError(t, err) {
					assert.Equal(t, s.want, v.([]string)[0])
				}
			}
		})
	}
}
