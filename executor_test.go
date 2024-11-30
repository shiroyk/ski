package ski

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidName(t *testing.T) {
	testsCases := []struct {
		name string
		ok   bool
	}{
		{"foo", true},
		{"foo.bar", true},
		{"_.bar_", true},
		{"_foo.bar_", true},
		{"foo123.bar123", true},
		{"123foo.bar123", true},
		{"foo.bar.baz", false},
		{"foo.bar.baz.", false},
		{"foo.bar.baz..", false},
		{"foo.bar.baz.", false},
	}

	for _, testCase := range testsCases {
		before, method, _ := strings.Cut(testCase.name, ".")
		ok := reName.MatchString(before)
		if method != "" {
			ok = ok && reName.MatchString(method)
		}
		assert.Equal(t, testCase.ok, ok, testCase.name)
	}
}
