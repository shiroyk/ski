package cloudcat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZeroOr(t *testing.T) {
	assert.Equal(t, 1, ZeroOr(0, 1))
}

func TestEmptyOr(t *testing.T) {
	assert.Equal(t, []int{1}, EmptyOr([]int{}, []int{1}))
}

func TestParseCookie(t *testing.T) {
	maxAge := "Name=Test;MaxAge=3600;"
	assert.Equal(t, "Name=Test; MaxAge=3600", CookieToString(ParseCookie(maxAge)))
}
