package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZeroOr(t *testing.T) {
	assert.Equal(t, ZeroOr(0, 1), 1)
}

func TestEmptyOr(t *testing.T) {
	assert.Equal(t, EmptyOr([]int{}, []int{1}), []int{1})
}

func TestParseCookie(t *testing.T) {
	maxAge := "MaxAge=3600;"
	assert.Equal(t, "MaxAge=3600", CookieToString(ParseCookie(maxAge)))
}
