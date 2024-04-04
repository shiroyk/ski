package ski

import (
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
		{"123foo.bar123", false},
		{"foo.bar.baz", false},
		{"foo.bar.baz.", false},
		{"foo.bar.baz..", false},
		{"foo.bar.baz.", false},
	}

	for _, testCase := range testsCases {
		assert.Equal(t, testCase.ok, isValidName(testCase.name), testCase.name)
	}
}
